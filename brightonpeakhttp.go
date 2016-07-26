package main

import (
	"log"
	"time"
)

const (
	BRIGHTONPEAK_API_URL       = "https://api.brightonpeak.com:8400"
	BRIGHTONPEAK_WEBSOCKET_URL = "wss://api.brightonpeak.com:8401"
)

type BrightonPeak struct {
	Name                        string
	Enabled                     bool
	Verbose                     bool
	Websocket                   bool
	RESTPollingDelay            time.Duration
	AuthenticatedAPISupport     bool
	APIKey, APISecret, ClientID string
	TakerFee, MakerFee          float64
	BaseCurrencies              []string
	AvailablePairs              []string
	EnabledPairs                []string
	API                         Alphapoint
}

func (b *BrightonPeak) SetDefaults() {
	b.Name = "Brighton Peak"
	b.Enabled = false
	b.TakerFee = 0.5
	b.MakerFee = 0.5
	b.Verbose = false
	b.Websocket = false
	b.RESTPollingDelay = 10
	b.API.APIUrl = BRIGHTONPEAK_API_URL
	b.API.WebsocketURL = BRIGHTONPEAK_WEBSOCKET_URL
}

func (b *BrightonPeak) GetName() string {
	return b.Name
}

func (b *BrightonPeak) SetEnabled(enabled bool) {
	b.Enabled = enabled
}

func (b *BrightonPeak) Setup(exch Exchanges) {
	if !exch.Enabled {
		b.SetEnabled(false)
	} else {
		b.Enabled = true
		b.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		b.SetAPIKeys(exch.APIKey, exch.APISecret, exch.ClientID)
		b.RESTPollingDelay = exch.RESTPollingDelay
		b.Verbose = exch.Verbose
		b.Websocket = exch.Websocket
		b.BaseCurrencies = SplitStrings(exch.BaseCurrencies, ",")
		b.AvailablePairs = SplitStrings(exch.AvailablePairs, ",")
		b.EnabledPairs = SplitStrings(exch.EnabledPairs, ",")
	}
}

func (k *BrightonPeak) GetEnabledCurrencies() []string {
	return k.EnabledPairs
}

func (b *BrightonPeak) Start() {
	go b.Run()
}

func (b *BrightonPeak) IsEnabled() bool {
	return b.Enabled
}

func (b *BrightonPeak) SetAPIKeys(apiKey, apiSecret, clientID string) {
	b.API.UserID = clientID
	b.API.APIKey = apiKey
	b.API.APISecret = apiSecret
}

func (b *BrightonPeak) GetFee(maker bool) float64 {
	if maker {
		return b.MakerFee
	} else {
		return b.TakerFee
	}
}

func (b *BrightonPeak) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", b.GetName(), IsEnabled(b.Websocket), BRIGHTONPEAK_WEBSOCKET_URL)
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	if b.Websocket {
		//go b.WebsocketClient()
	}

	exchangeProducts, err := b.GetProductPairs()
	if err != nil || !exchangeProducts.IsAccepted {
		log.Printf("%s Failed to get available products.\n", b.GetName())
	} else {
		currencies := []string{}
		for _, x := range exchangeProducts.ProductPairs {
			currencies = append(currencies, x.Name)
		}
		diff := StringSliceDifference(b.AvailablePairs, currencies)
		if len(diff) > 0 {
			exch, err := GetExchangeConfig(b.Name)
			if err != nil {
				log.Println(err)
			} else {
				log.Printf("%s Updating available pairs. Difference: %s.\n", b.Name, diff)
				exch.AvailablePairs = JoinStrings(currencies, ",")
				UpdateExchangeConfig(exch)
			}
		}
	}

	for b.Enabled {
		for _, x := range b.EnabledPairs {
			ticker, err := b.GetTicker(x)

			if err != nil {
				log.Println(err)
				continue
			}
			log.Printf("%s %s Last %f High %f Low %f Volume %f\n", b.GetName(), x, ticker.Last, ticker.High, ticker.Low, ticker.Volume)
			AddExchangeInfo(b.GetName(), x[0:3], x[3:], ticker.Last, 0)
		}
		time.Sleep(time.Second * b.RESTPollingDelay)
	}
}

func (b *BrightonPeak) GetTicker(symbol string) (AlphapointTicker, error) {
	return b.API.GetTicker(symbol)
}

func (b *BrightonPeak) GetTickerPrice(currency string) TickerPrice {
	var tickerPrice TickerPrice
	ticker, err := b.GetTicker(currency)
	if err != nil {
		log.Println(err)
		return tickerPrice
	}
	tickerPrice.Ask = ticker.Ask
	tickerPrice.Bid = ticker.Bid
	tickerPrice.CryptoCurrency = currency
	tickerPrice.Low = ticker.Low
	tickerPrice.Last = ticker.Last
	tickerPrice.Volume = ticker.Volume
	tickerPrice.High = ticker.High

	return tickerPrice
}

func (b *BrightonPeak) GetTrades(symbol string, startIndex, count int) (AlphapointTrades, error) {
	return b.API.GetTrades(symbol, startIndex, count)
}

func (b *BrightonPeak) GetTradesByDate(symbol string, startIndex, count int) (AlphapointTrades, error) {
	return b.API.GetTrades(symbol, startIndex, count)
}

func (b *BrightonPeak) GetOrderBook(symbol string) (AlphapointOrderbook, error) {
	return b.API.GetOrderbook(symbol)
}

func (b *BrightonPeak) GetProductPairs() (AlphapointProductPairs, error) {
	return b.API.GetProductPairs()
}

func (b *BrightonPeak) GetProducts() (AlphapointProducts, error) {
	return b.API.GetProducts()
}

func (b *BrightonPeak) GreateAccount(firstName, lastName, email, phone, password string) error {
	return b.API.CreateAccount(firstName, lastName, email, phone, password)
}

func (b *BrightonPeak) GetUserInfo() (AlphapointUserInfo, error) {
	return b.API.GetUserInfo()
}

func (b *BrightonPeak) SetUserInfo() {} // to-do

func (b *BrightonPeak) GetAccountInfo() (AlphapointAccountInfo, error) {
	return b.API.GetAccountInfo()
}

func (b *BrightonPeak) GetAccountTrades(symbol string, startIndex, count int) (AlphapointTrades, error) {
	return b.API.GetAccountTrades(symbol, startIndex, count)
}

func (b *BrightonPeak) GetDepositAddresses() ([]AlphapointDepositAddresses, error) {
	return b.API.GetDepositAddresses()
}

func (b *BrightonPeak) WithdrawCoins(symbol, product string, amount float64, address string) error {
	return b.API.WithdrawCoins(symbol, product, amount, address)
}

func (b *BrightonPeak) CreateOrder(symbol, side string, orderType int, quantity, price float64) (int64, error) {
	return b.API.CreateOrder(symbol, side, orderType, quantity, price)
}

func (b *BrightonPeak) ModifyOrder(symbol string, OrderID, action int64) (int64, error) {
	return b.API.ModifyOrder(symbol, OrderID, action)
}

func (b *BrightonPeak) CancelOrder(symbol string, OrderID int64) (int64, error) {
	return b.API.CancelOrder(symbol, OrderID)
}

func (b *BrightonPeak) CancelAllOrders(symbol string) error {
	return b.API.CancelAllOrders(symbol)
}

func (b *BrightonPeak) GetOrders() ([]AlphapointOpenOrders, error) {
	return b.API.GetOrders()
}

func (b *BrightonPeak) GetOrderFee(symbol, side string, quantity, price float64) (float64, error) {
	return b.API.GetOrderFee(symbol, side, quantity, price)
}
