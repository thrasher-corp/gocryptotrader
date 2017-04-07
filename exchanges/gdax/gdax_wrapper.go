package gdax

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
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
		currencies := []string{}
		for _, x := range exchangeProducts {
			if x.ID != "BTC" && x.ID != "USD" && x.ID != "GBP" {
				currencies = append(currencies, x.ID[0:3]+x.ID[4:])
			}
		}
		err = g.UpdateAvailableCurrencies(currencies)
		if err != nil {
			log.Printf("%s Failed to get config.\n", g.GetName())
		}
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
				stats.AddExchangeInfo(g.GetName(), currency[0:3], currency[4:], ticker.Last, ticker.Volume)
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
		exchangeCurrency.TotalValue = accountBalance[i].Available
		exchangeCurrency.Hold = accountBalance[i].Hold

		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}

func (g *GDAX) GetTickerPrice(currency string) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(g.GetName(), currency[0:3], currency[4:])
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

func (g *GDAX) GetOrderbookEx(currency string) (orderbook.OrderbookBase, error) {
	ob, err := orderbook.GetOrderbook(g.GetName(), currency[0:3], currency[4:])
	if err == nil {
		return ob, nil
	}

	var orderBook orderbook.OrderbookBase
	orderbookNew, err := g.GetOrderbook(currency, 2)
	if err != nil {
		return orderBook, err
	}

	obNew := orderbookNew.(GDAXOrderbookL1L2)

	for x, _ := range obNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.OrderbookItem{Amount: obNew.Bids[x].Amount, Price: obNew.Bids[x].Price})
	}

	for x, _ := range obNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.OrderbookItem{Amount: obNew.Bids[x].Amount, Price: obNew.Bids[x].Price})
	}
	orderBook.FirstCurrency = currency[0:3]
	orderBook.SecondCurrency = currency[4:]
	orderbook.ProcessOrderbook(g.GetName(), orderBook.FirstCurrency, orderBook.SecondCurrency, orderBook)
	return orderBook, nil
}
