package gemini

import (
	"log"
	"net/url"
	"time"

	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
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
		err = g.UpdateAvailableCurrencies(exchangeProducts)
		if err != nil {
			log.Printf("%s Failed to get config.\n", g.GetName())
		}
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
				stats.AddExchangeInfo(g.GetName(), currency[0:3], currency[3:], ticker.Last, ticker.Volume)
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

func (g *Gemini) GetOrderbookEx(currency string) (orderbook.OrderbookBase, error) {
	ob, err := orderbook.GetOrderbook(g.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return ob, nil
	}

	var orderBook orderbook.OrderbookBase
	orderbookNew, err := g.GetOrderbook(currency, url.Values{})
	if err != nil {
		return orderBook, err
	}

	for x, _ := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.OrderbookItem{Amount: orderbookNew.Bids[x].Amount, Price: orderbookNew.Bids[x].Price})
	}

	for x, _ := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.OrderbookItem{Amount: orderbookNew.Asks[x].Amount, Price: orderbookNew.Asks[x].Price})
	}

	orderBook.FirstCurrency = currency[0:3]
	orderBook.SecondCurrency = currency[3:]
	orderbook.ProcessOrderbook(g.GetName(), orderBook.FirstCurrency, orderBook.SecondCurrency, orderBook)
	return orderBook, nil
}
