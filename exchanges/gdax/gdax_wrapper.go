package gdax

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (g *GDAX) Start() {
	go g.Run()
}

func (g *GDAX) Run() {
	if g.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", g.GetName(), common.IsEnabled(g.Websocket), GDAX_WEBSOCKET_URL)
		log.Printf("%s polling delay: %ds.\n", g.GetName(), g.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", g.GetName(), len(g.EnabledPairs), g.EnabledPairs)
	}

	if g.Websocket {
		go g.WebsocketClient()
	}

	exchangeProducts, err := g.GetProducts()
	if err != nil {
		log.Printf("%s Failed to get available products.\n", g.GetName())
	} else {
		log.Println(exchangeProducts)
		/*
			currencies := []string{}
			for _, x := range exchangeProducts {
				if x.ID != "BTC" && x.ID != "USD" && x.ID != "GBP" {
					currencies = append(currencies, x.ID[0:3]+x.ID[4:])
				}
			}
			diff := common.StringSliceDifference(g.AvailablePairs, currencies)
			if len(diff) > 0 {
				exch, err := bot.config.GetExchangeConfig(g.Name)
				if err != nil {
					log.Println(err)
				} else {
					log.Printf("%s Updating available pairs. Difference: %s.\n", g.Name, diff)
					exch.AvailablePairs = common.JoinStrings(currencies, ",")
					bot.config.UpdateExchangeConfig(exch)
				}
			}
		*/
	}

	for g.Enabled {
		for _, x := range g.EnabledPairs {
			currency := x[0:3] + "-" + x[3:]
			go func() {
				ticker, err := g.GetTickerPrice(currency)

				if err != nil {
					log.Println(err)
					return
				}
				log.Printf("GDAX %s: Last %f High %f Low %f Volume %f\n", currency, ticker.Last, ticker.High, ticker.Low, ticker.Volume)
				//AddExchangeInfo(g.GetName(), currency[0:3], currency[4:], ticker.Last, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * g.RESTPollingDelay)
	}
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the GDAX exchange
func (e *GDAX) GetExchangeAccountInfo() (exchange.ExchangeAccountInfo, error) {
	var response exchange.ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	accountBalance, err := e.GetAccounts()
	if err != nil {
		return response, err
	}
	for i := 0; i < len(accountBalance); i++ {
		var exchangeCurrency exchange.ExchangeAccountCurrencyInfo
		exchangeCurrency.CurrencyName = accountBalance[i].Currency
		exchangeCurrency.TotalValue = accountBalance[i].Balance
		exchangeCurrency.Hold = accountBalance[i].Hold

		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}

func (g *GDAX) GetTickerPrice(currency string) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(g.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return tickerNew, nil
	}

	var tickerPrice ticker.TickerPrice
	tick, err := g.GetTicker(currency)
	if err != nil {
		return ticker.TickerPrice{}, err
	}

	stats, err := g.GetStats(currency)

	if err != nil {
		return ticker.TickerPrice{}, err
	}

	tickerPrice.FirstCurrency = currency[0:3]
	tickerPrice.SecondCurrency = currency[4:]
	tickerPrice.Volume = stats.Volume
	tickerPrice.Last = tick.Price
	tickerPrice.High = stats.High
	tickerPrice.Low = stats.Low
	ticker.ProcessTicker(g.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}
