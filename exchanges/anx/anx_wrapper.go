package anx

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (a *ANX) Start() {
	go a.Run()
}

func (a *ANX) Run() {
	if a.Verbose {
		log.Printf("%s polling delay: %ds.\n", a.GetName(), a.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", a.GetName(), len(a.EnabledPairs), a.EnabledPairs)
	}

	for a.Enabled {
		for _, x := range a.EnabledPairs {
			currency := pair.NewCurrencyPair(x[0:3], x[3:])
			go func() {
				ticker, err := a.GetTickerPrice(currency)
				if err != nil {
					log.Println(err)
					return
				}
				log.Printf("ANX %s: Last %f High %f Low %f Volume %f\n", currency.Pair(), ticker.Last, ticker.High, ticker.Low, ticker.Volume)
				stats.AddExchangeInfo(a.GetName(), currency.GetFirstCurrency().String(), currency.GetSecondCurrency().String(), ticker.Last, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * a.RESTPollingDelay)
	}
}

func (a *ANX) GetTickerPrice(p pair.CurrencyPair) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(a.GetName(), p)
	if err == nil {
		return tickerNew, nil
	}

	var tickerPrice ticker.TickerPrice
	tick, err := a.GetTicker(p.Pair().String())
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = p
	tickerPrice.Ask = tick.Data.Buy.Value
	tickerPrice.Bid = tick.Data.Sell.Value
	tickerPrice.Low = tick.Data.Low.Value
	tickerPrice.Last = tick.Data.Last.Value
	tickerPrice.Volume = tick.Data.Vol.Value
	tickerPrice.High = tick.Data.High.Value
	ticker.ProcessTicker(a.GetName(), p, tickerPrice)
	return tickerPrice, nil
}

func (e *ANX) GetOrderbookEx(p pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	return orderbook.OrderbookBase{}, nil
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the ANX exchange
func (e *ANX) GetExchangeAccountInfo() (exchange.ExchangeAccountInfo, error) {
	var response exchange.ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	return response, nil
}
