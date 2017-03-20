package liqui

import (
	"errors"
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (l *Liqui) Start() {
	go l.Run()
}

func (l *Liqui) Run() {
	if l.Verbose {
		log.Printf("%s polling delay: %ds.\n", l.GetName(), l.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", l.GetName(), len(l.EnabledPairs), l.EnabledPairs)
	}

	var err error
	l.Info, err = l.GetInfo()
	if err != nil {
		log.Printf("%s Unable to fetch info.\n", l.GetName())
	} else {
		exchangeProducts := l.GetAvailablePairs(true)
		log.Println(exchangeProducts)
		/*
			diff := common.StringSliceDifference(l.AvailablePairs, exchangeProducts)
			if len(diff) > 0 {
				exch, err := bot.config.GetExchangeConfig(l.Name)
				if err != nil {
					log.Println(err)
				} else {
					log.Printf("%s Updating available pairs. Difference: %s.\n", l.Name, diff)
					exch.AvailablePairs = common.JoinStrings(exchangeProducts, ",")
					bot.config.UpdateExchangeConfig(exch)
				}
			}
		*/
	}

	pairs := []string{}
	for _, x := range l.EnabledPairs {
		currencies := common.SplitStrings(x, "_")
		x = common.StringToLower(currencies[0]) + "_" + common.StringToLower(currencies[1])
		pairs = append(pairs, x)
	}
	pairsString := common.JoinStrings(pairs, "-")

	for l.Enabled {
		go func() {
			ticker, err := l.GetTicker(pairsString)
			if err != nil {
				log.Println(err)
				return
			}
			for x, y := range ticker {
				//currencies := common.SplitStrings(x, "_")
				x = common.StringToUpper(x)
				log.Printf("Liqui %s: Last %f High %f Low %f Volume %f\n", x, y.Last, y.High, y.Low, y.Vol_cur)
				l.Ticker[x] = y
				//AddExchangeInfo(l.GetName(), common.StringToUpper(currencies[0]), common.StringToUpper(currencies[1]), y.Last, y.Vol_cur)
			}
		}()
		time.Sleep(time.Second * l.RESTPollingDelay)
	}
}

func (l *Liqui) GetTickerPrice(currency string) (ticker.TickerPrice, error) {
	var tickerPrice ticker.TickerPrice
	tick, ok := l.Ticker[currency]
	if !ok {
		return tickerPrice, errors.New("Unable to get currency.")
	}
	tickerPrice.Ask = tick.Buy
	tickerPrice.Bid = tick.Sell
	currencies := common.SplitStrings(currency, "_")
	tickerPrice.FirstCurrency = currencies[0]
	tickerPrice.SecondCurrency = currencies[1]
	tickerPrice.Low = tick.Low
	tickerPrice.Last = tick.Last
	tickerPrice.Volume = tick.Vol_cur
	tickerPrice.High = tick.High
	ticker.ProcessTicker(l.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the Liqui exchange
func (e *Liqui) GetExchangeAccountInfo() (exchange.ExchangeAccountInfo, error) {
	var response exchange.ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	accountBalance, err := e.GetAccountInfo()
	if err != nil {
		return response, err
	}

	for x, y := range accountBalance.Funds {
		var exchangeCurrency exchange.ExchangeAccountCurrencyInfo
		exchangeCurrency.CurrencyName = common.StringToUpper(x)
		exchangeCurrency.TotalValue = y
		exchangeCurrency.Hold = 0
		response.Currencies = append(response.Currencies, exchangeCurrency)
	}

	return response, nil
}
