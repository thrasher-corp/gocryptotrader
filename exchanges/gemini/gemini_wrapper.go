package gemini

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (g *Gemini) Start() {
	go g.Run()
}

func (g *Gemini) Run() {
	if g.Verbose {
		log.Printf("%s polling delay: %ds.\n", g.GetName(), g.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", g.GetName(), len(g.EnabledPairs), g.EnabledPairs)
	}

	exchangeProducts, err := g.GetSymbols()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", g.GetName())
	} else {
		log.Println(exchangeProducts)
		/*
			exchangeProducts = common.SplitStrings(common.StringToUpper(common.JoinStrings(exchangeProducts, ",")), ",")
			diff := common.StringSliceDifference(g.AvailablePairs, exchangeProducts)
			if len(diff) > 0 {
				exch, err := bot.config.GetExchangeConfig(g.Name)
				if err != nil {
					log.Println(err)
				} else {
					log.Printf("%s Updating available pairs. Difference: %s.\n", g.Name, diff)
					exch.AvailablePairs = common.JoinStrings(exchangeProducts, ",")
					bot.config.UpdateExchangeConfig(exch)
				}
			}
		*/
	}

	for g.Enabled {
		for _, x := range g.EnabledPairs {
			currency := x
			go func() {
				ticker, err := g.GetTickerPrice(currency)
				if err != nil {
					log.Println(err)
					return
				}
				log.Printf("Gemini %s Last %f Bid %f Ask %f Volume %f\n", currency, ticker.Last, ticker.Bid, ticker.Ask, ticker.Volume)
				//AddExchangeInfo(g.GetName(), currency[0:3], currency[3:], ticker.Last, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * g.RESTPollingDelay)
	}
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the Gemini exchange
func (e *Gemini) GetExchangeAccountInfo() (exchange.ExchangeAccountInfo, error) {
	var response exchange.ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	accountBalance, err := e.GetBalances()
	if err != nil {
		return response, err
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

func (g *Gemini) GetTickerPrice(currency string) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(g.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return tickerNew, nil
	}

	var tickerPrice ticker.TickerPrice
	tick, err := g.GetTicker(currency)
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Ask = tick.Ask
	tickerPrice.Bid = tick.Bid
	tickerPrice.FirstCurrency = currency[0:3]
	tickerPrice.SecondCurrency = currency[3:]
	tickerPrice.Last = tick.Last
	tickerPrice.Volume = tick.Volume.USD
	ticker.ProcessTicker(g.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}
