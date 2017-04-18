package gdax

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
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
			currency := pair.NewCurrencyPair(x[0:3], x[3:])
			currency.Delimiter = "-"
			go func() {
				ticker, err := g.GetTickerPrice(currency)

				if err != nil {
					log.Println(err)
					return
				}
				log.Printf("GDAX %s: Last %f High %f Low %f Volume %f\n", currency.Pair().String(), ticker.Last, ticker.High, ticker.Low, ticker.Volume)
				stats.AddExchangeInfo(g.GetName(), currency.GetFirstCurrency().String(), currency.GetSecondCurrency().String(), ticker.Last, ticker.Volume)
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

func (g *GDAX) GetTickerPrice(p pair.CurrencyPair) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(g.GetName(), p)
	if err == nil {
		return tickerNew, nil
	}

	var tickerPrice ticker.TickerPrice
	tick, err := g.GetTicker(p.Pair().String())
	if err != nil {
		return ticker.TickerPrice{}, err
	}

	stats, err := g.GetStats(p.Pair().String())

	if err != nil {
		return ticker.TickerPrice{}, err
	}

	tickerPrice.Pair = p
	tickerPrice.Volume = stats.Volume
	tickerPrice.Last = tick.Price
	tickerPrice.High = stats.High
	tickerPrice.Low = stats.Low
	ticker.ProcessTicker(g.GetName(), p, tickerPrice)
	return tickerPrice, nil
}

func (g *GDAX) GetOrderbookEx(p pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	ob, err := orderbook.GetOrderbook(g.GetName(), p)
	if err == nil {
		return ob, nil
	}

	var orderBook orderbook.OrderbookBase
	orderbookNew, err := g.GetOrderbook(p.Pair().String(), 2)
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
	orderBook.Pair = p
	orderbook.ProcessOrderbook(g.GetName(), p, orderBook)
	return orderBook, nil
}
