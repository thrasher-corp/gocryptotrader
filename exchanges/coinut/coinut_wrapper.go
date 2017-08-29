package coinut

import (
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (c *COINUT) Start() {
	go c.Run()
}

func (c *COINUT) Run() {
	if c.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", c.GetName(), common.IsEnabled(c.Websocket), COINUT_WEBSOCKET_URL)
		log.Printf("%s polling delay: %ds.\n", c.GetName(), c.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", c.GetName(), len(c.EnabledPairs), c.EnabledPairs)
	}

	if c.Websocket {
		go c.WebsocketClient()
	}

	exchangeProducts, err := c.GetInstruments()
	if err != nil {
		log.Printf("%s Failed to get available products.\n", c.GetName())
		return
	}

	currencies := []string{}
	c.InstrumentMap = make(map[string]int)
	for x, y := range exchangeProducts.Instruments {
		c.InstrumentMap[x] = y[0].InstID
		currencies = append(currencies, x)
	}

	err = c.UpdateAvailableCurrencies(currencies, false)
	if err != nil {
		log.Printf("%s Failed to get config.\n", c.GetName())
	}
}

// GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the COINUT exchange
func (c *COINUT) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	/*
		response.ExchangeName = e.GetName()
		accountBalance, err := e.GetAccounts()
		if err != nil {
			return response, err
		}
		for i := 0; i < len(accountBalance); i++ {
			var exchangeCurrency exchange.AccountCurrencyInfo
			exchangeCurrency.CurrencyName = accountBalance[i].Currency
			exchangeCurrency.TotalValue = accountBalance[i].Available
			exchangeCurrency.Hold = accountBalance[i].Hold

			response.Currencies = append(response.Currencies, exchangeCurrency)
		}
	*/
	return response, nil
}

func (c *COINUT) UpdateTicker(p pair.CurrencyPair) (ticker.TickerPrice, error) {
	var tickerPrice ticker.TickerPrice
	tick, err := c.GetInstrumentTicker(c.InstrumentMap[p.Pair().String()])
	if err != nil {
		return ticker.TickerPrice{}, err
	}

	tickerPrice.Pair = p
	tickerPrice.Volume = tick.Volume
	tickerPrice.Last = tick.Last
	tickerPrice.High = tick.HighestBuy
	tickerPrice.Low = tick.LowestSell
	ticker.ProcessTicker(c.GetName(), p, tickerPrice)
	return tickerPrice, nil

}

func (c *COINUT) GetTickerPrice(p pair.CurrencyPair) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(c.GetName(), p)
	if err != nil {
		return c.UpdateTicker(p)
	}
	return tickerNew, nil
}

func (c *COINUT) GetOrderbookEx(p pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	ob, err := orderbook.GetOrderbook(c.GetName(), p)
	if err == nil {
		return ob, nil
	}

	var orderBook orderbook.OrderbookBase
	orderbookNew, err := c.GetInstrumentOrderbook(c.InstrumentMap[p.Pair().String()], 200)
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Buy {
		orderBook.Bids = append(orderBook.Bids, orderbook.OrderbookItem{Amount: orderbookNew.Buy[x].Quantity, Price: orderbookNew.Buy[x].Price})
	}

	for x := range orderbookNew.Sell {
		orderBook.Asks = append(orderBook.Asks, orderbook.OrderbookItem{Amount: orderbookNew.Sell[x].Quantity, Price: orderbookNew.Sell[x].Price})
	}
	orderBook.Pair = p
	orderbook.ProcessOrderbook(c.GetName(), p, orderBook)
	return orderBook, nil
}
