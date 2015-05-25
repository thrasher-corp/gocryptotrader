package main

import (
	"log"
	"time"
)

const (
	DWVX_API_URL = "https://api.dwvx.com.au:8400"
)

type DWVX struct {
	Name                        string
	Enabled                     bool
	Verbose                     bool
	Websocket                   bool
	RESTPollingDelay            time.Duration
	AuthenticatedAPISupport     bool
	ClientID, APIKey, APISecret string
	Ticker                      AlphapointTicker
	TakerFee, MakerFee          float64
	BaseCurrencies              []string
	AvailablePairs              []string
	EnabledPairs                []string
	API                         Alphapoint
	DepositAddresses            map[string]string
}

func (d *DWVX) SetDefaults() {
	d.Name = "DWVX"
	d.API.APIUrl = DWVX_API_URL
	d.Enabled = true
	d.Verbose = false
	d.Websocket = false
	d.RESTPollingDelay = 10
	d.DepositAddresses = make(map[string]string)
}

func (d *DWVX) GetName() string {
	return d.Name
}

func (d *DWVX) SetEnabled(enabled bool) {
	d.Enabled = enabled
}

func (d *DWVX) IsEnabled() bool {
	return d.Enabled
}

func (d *DWVX) SetAPIKeys(userID, apiKey, apiSecret string) {
	d.API.APIKey = apiKey
	d.API.APISecret = apiSecret
	d.API.UserID = userID
}

func (d *DWVX) Run() {
	if d.Verbose {
		log.Printf("%s polling delay: %ds.\n", d.GetName(), d.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", d.GetName(), len(d.EnabledPairs), d.EnabledPairs)
	}

	products, err := d.GetProductPairs()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", d.GetName())
	} else {
		availProducts := []string{}
		for _, x := range products.ProductPairs {
			availProducts = append(availProducts, x.Name)
		}

		diff := StringSliceDifference(d.AvailablePairs, availProducts)
		if len(diff) > 0 {
			exch, err := GetExchangeConfig(d.Name)
			if err != nil {
				log.Println(err)
			} else {
				log.Printf("%s Updating available pairs. Difference: %s.\n", d.Name, diff)
				exch.AvailablePairs = JoinStrings(availProducts, ",")
				UpdateExchangeConfig(exch)
			}
		}
	}

	for d.Enabled {
		for _, x := range d.EnabledPairs {
			currency := x
			go func() {
				ticker, err := d.GetTicker(currency)
				if err != nil {
					log.Println(err)
					return
				}
				log.Printf("DWVX %s: Last %f High %f Low %f Volume %f\n", currency, ticker.Last, ticker.High, ticker.Low, ticker.Total24HrQtyTraded)
				AddExchangeInfo(d.GetName(), currency[0:3], currency[3:], ticker.Last, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * d.RESTPollingDelay)
	}
}

func (d *DWVX) GetTicker(symbol string) (AlphapointTicker, error) {
	return d.API.GetTicker(symbol)
}

func (d *DWVX) GetTrades(symbol string, startIndex, count int) (AlphapointTrades, error) {
	return d.API.GetTrades(symbol, startIndex, count)
}

func (d *DWVX) GetTradesByDate(symbol string, startDate, endDate int64) (AlphapointTradesByDate, error) {
	return d.API.GetTradesByDate(symbol, startDate, endDate)
}

func (d *DWVX) GetOrderbook(symbol string) (AlphapointOrderbook, error) {
	return d.API.GetOrderbook(symbol)
}

func (d *DWVX) GetProductPairs() (AlphapointProductPairs, error) {
	return d.API.GetProductPairs()
}

func (d *DWVX) GetProducts() (AlphapointProducts, error) {
	return d.API.GetProducts()
}

func (d *DWVX) GetUserInfo() (AlphapointUserInfo, error) {
	return d.API.GetUserInfo()
}

func (d *DWVX) GetAccountTrades(symbol string, startIndex, count int) (AlphapointTrades, error) {
	return d.API.GetAccountTrades(symbol, startIndex, count)
}
func (d *DWVX) GetAccountInfo() (AlphapointAccountInfo, error) {
	return d.API.GetAccountInfo()
}

func (d *DWVX) GetDepositAddresses() error {
	result, err := d.API.GetDepositAddresses()
	if err != nil {
		return err
	}
	for _, x := range result {
		if x.DepositAddress != "" {
			d.DepositAddresses[x.Name] = x.DepositAddress
		}
	}
	return nil
}

func (d *DWVX) WithdrawCoins(symbol, product string, amount float64, address string) error {
	return d.API.WithdrawCoins(symbol, product, amount, address)
}

func (d *DWVX) CreateOrder(symbol, side string, orderType int, quantity, price float64) (int64, error) {
	return d.API.CreateOrder(symbol, side, orderType, quantity, price)
}

func (d *DWVX) ModifyOrder(symbol string, OrderID, action int64) (int64, error) {
	return d.API.ModifyOrder(symbol, OrderID, action)
}

func (d *DWVX) CancelOrder(symbol string, orderID int64) (int64, error) {
	return d.API.CancelOrder(symbol, orderID)
}

func (d *DWVX) CancelAllOrders(symbol string) error {
	return d.API.CancelAllOrders(symbol)
}

func (d *DWVX) GetOrders() ([]AlphapointOpenOrders, error) {
	return d.API.GetOrders()
}

func (d *DWVX) GetOrderFee(symbol, side string, amount, price float64) (float64, error) {
	return d.API.GetOrderFee(symbol, side, amount, price)
}
