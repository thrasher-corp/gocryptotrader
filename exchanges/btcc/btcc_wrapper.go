package btcc

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (b *BTCC) Start() {
	go b.Run()
}

func (b *BTCC) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s.", b.GetName(), common.IsEnabled(b.Websocket))
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	if b.Websocket {
		go b.WebsocketClient()
	}

	for b.Enabled {
		for _, x := range b.EnabledPairs {
			currency := x
			go func() {
				ticker, err := b.GetTickerPrice(common.StringToLower(currency))
				if err != nil {
					log.Println(err)
					return
				}
				log.Printf("BTCC %s: Last %f High %f Low %f Volume %f\n", currency, ticker.Last, ticker.High, ticker.Low, ticker.Volume)
				stats.AddExchangeInfo(b.GetName(), currency[0:3], currency[3:], ticker.Last, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * b.RESTPollingDelay)
	}
}

func (b *BTCC) GetTickerPrice(currency string) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return tickerNew, nil
	}

	var tickerPrice ticker.TickerPrice
	tick, err := b.GetTicker(currency)
	tickerPrice.Ask = tick.Sell
	tickerPrice.Bid = tick.Buy
	tickerPrice.FirstCurrency = currency[0:3]
	tickerPrice.SecondCurrency = currency[3:]
	tickerPrice.Low = tick.Low
	tickerPrice.Last = tick.Last
	tickerPrice.Volume = tick.Vol
	tickerPrice.High = tick.High
	ticker.ProcessTicker(b.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}

func (b *BTCC) GetOrderbookEx(currency string) (orderbook.OrderbookBase, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return ob, nil
	}

	var orderBook orderbook.OrderbookBase
	orderbookNew, err := b.GetOrderBook(currency, 100)
	if err != nil {
		return orderBook, err
	}

	for x, _ := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(ob.Bids, orderbook.OrderbookItem{Price: data[0], Amount: data[1]})
	}

	for x, _ := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(ob.Asks, orderbook.OrderbookItem{Price: data[0], Amount: data[1]})
	}

	orderBook.FirstCurrency = currency[0:3]
	orderBook.SecondCurrency = currency[3:]
	orderbook.ProcessOrderbook(b.GetName(), orderBook.FirstCurrency, orderBook.SecondCurrency, orderBook)
	return orderBook, nil
}

//TODO: Retrieve BTCC info
//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the Kraken exchange
func (e *BTCC) GetExchangeAccountInfo() (exchange.ExchangeAccountInfo, error) {
	var response exchange.ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	return response, nil
}
