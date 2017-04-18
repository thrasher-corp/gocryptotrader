package localbitcoins

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (l *LocalBitcoins) Start() {
	go l.Run()
}

func (l *LocalBitcoins) Run() {
	if l.Verbose {
		log.Printf("%s polling delay: %ds.\n", l.GetName(), l.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", l.GetName(), len(l.EnabledPairs), l.EnabledPairs)
	}

	for l.Enabled {
		for _, x := range l.EnabledPairs {
			currency := pair.NewCurrencyPair("BTC", x[3:])
			ticker, err := l.GetTickerPrice(currency)

			if err != nil {
				log.Println(err)
				return
			}

			log.Printf("LocalBitcoins BTC %s: Last %f Volume %f\n", currency.Pair().String(), ticker.Last, ticker.Volume)
			stats.AddExchangeInfo(l.GetName(), currency.GetFirstCurrency().String(), currency.GetSecondCurrency().String(), ticker.Last, ticker.Volume)
		}
		time.Sleep(time.Second * l.RESTPollingDelay)
	}
}

func (l *LocalBitcoins) GetTickerPrice(p pair.CurrencyPair) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(l.GetName(), p)
	if err == nil {
		return tickerNew, nil
	}

	tick, err := l.GetTicker()
	if err != nil {
		return ticker.TickerPrice{}, err
	}

	var tickerPrice ticker.TickerPrice
	for key, value := range tick {
		tickerPrice.Pair = p
		tickerPrice.Last = value.Rates.Last
		tickerPrice.Pair.SecondCurrency = pair.CurrencyItem(key)
		tickerPrice.Volume = value.VolumeBTC
		ticker.ProcessTicker(l.GetName(), p, tickerPrice)
	}
	return tickerPrice, nil
}

func (l *LocalBitcoins) GetOrderbookEx(p pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	return orderbook.OrderbookBase{}, nil
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the LocalBitcoins exchange
func (e *LocalBitcoins) GetExchangeAccountInfo() (exchange.ExchangeAccountInfo, error) {
	var response exchange.ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	accountBalance, err := e.GetWalletBalance()
	if err != nil {
		return response, err
	}
	var exchangeCurrency exchange.ExchangeAccountCurrencyInfo
	exchangeCurrency.CurrencyName = "BTC"
	exchangeCurrency.TotalValue = accountBalance.Total.Balance

	response.Currencies = append(response.Currencies, exchangeCurrency)
	return response, nil
}
