package bitfinex

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (b *Bitfinex) Start() {
	go b.Run()
}

func (b *Bitfinex) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s.", b.GetName(), common.IsEnabled(b.Websocket))
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	if b.Websocket {
		go b.WebsocketClient()
	}

	exchangeProducts, err := b.GetSymbols()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", b.GetName())
	} else {
		log.Println(exchangeProducts)
		/*
			exchangeProducts = common.SplitStrings(common.StringToUpper(common.JoinStrings(exchangeProducts, ",")), ",")
			diff := common.StringSliceDifference(b.AvailablePairs, exchangeProducts)
			if len(diff) > 0 {
				exch, err := bot.config.GetExchangeConfig(b.Name)
				if err != nil {
					log.Println(err)
				} else {
					log.Printf("%s Updating available pairs. Difference: %s.\n", b.Name, diff)
					exch.AvailablePairs = common.JoinStrings(exchangeProducts, ",")
					bot.config.UpdateExchangeConfig(exch)
				}
			}
		*/
	}

	for b.Enabled {
		for _, x := range b.EnabledPairs {
			currency := x
			go func() {
				ticker, err := b.GetTickerPrice(currency)
				if err != nil {
					return
				}
				log.Printf("Bitfinex %s Last %f High %f Low %f Volume %f\n", currency, ticker.Last, ticker.High, ticker.Low, ticker.Volume)
				//AddExchangeInfo(b.GetName(), currency[0:3], currency[3:], ticker.Last, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * b.RESTPollingDelay)
	}
}

func (b *Bitfinex) GetTickerPrice(currency string) (ticker.TickerPrice, error) {
	tick, err := ticker.GetTicker(b.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return tick, nil
	}

	var tickerPrice ticker.TickerPrice
	tickerNew, err := b.GetTicker(currency, nil)
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Ask = tickerNew.Ask
	tickerPrice.Bid = tickerNew.Bid
	tickerPrice.FirstCurrency = currency[0:3]
	tickerPrice.SecondCurrency = currency[3:]
	tickerPrice.Low = tickerNew.Low
	tickerPrice.Last = tickerNew.Last
	tickerPrice.Volume = tickerNew.Volume
	tickerPrice.High = tickerNew.High
	ticker.ProcessTicker(b.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the Bitfinex exchange
func (e *Bitfinex) GetExchangeAccountInfo() (exchange.ExchangeAccountInfo, error) {
	var response exchange.ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	accountBalance, err := e.GetAccountBalance()
	if err != nil {
		return response, err
	}
	if !e.Enabled {
		return response, nil
	}

	for i := 0; i < len(accountBalance); i++ {
		var exchangeCurrency exchange.ExchangeAccountCurrencyInfo
		exchangeCurrency.CurrencyName = accountBalance[i].Currency
		exchangeCurrency.TotalValue = accountBalance[i].Amount
		exchangeCurrency.Hold = accountBalance[i].Available

		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}
