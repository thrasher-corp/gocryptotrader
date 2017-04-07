package lakebtc

import (
	"log"
	"strconv"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (l *LakeBTC) Start() {
	go l.Run()
}
func (l *LakeBTC) Run() {
	if l.Verbose {
		log.Printf("%s polling delay: %ds.\n", l.GetName(), l.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", l.GetName(), len(l.EnabledPairs), l.EnabledPairs)
	}

	for l.Enabled {
		for _, x := range l.EnabledPairs {
			ticker, err := l.GetTickerPrice(x)
			if err != nil {
				log.Println(err)
				continue
			}
			log.Printf("LakeBTC BTC %s: Last %f High %f Low %f Volume %f\n", x[3:], ticker.Last, ticker.High, ticker.Low, ticker.Volume)
			stats.AddExchangeInfo(l.GetName(), x[0:3], x[3:], ticker.Last, ticker.Volume)
		}
		time.Sleep(time.Second * l.RESTPollingDelay)
	}
}

func (l *LakeBTC) GetTickerPrice(currency string) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(l.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return tickerNew, nil
	}

	tick, err := l.GetTicker()
	if err != nil {
		return ticker.TickerPrice{}, err
	}

	result, ok := tick[currency]
	if !ok {
		return ticker.TickerPrice{}, err
	}

	var tickerPrice ticker.TickerPrice
	tickerPrice.Ask = result.Ask
	tickerPrice.Bid = result.Bid
	tickerPrice.Volume = result.Volume
	tickerPrice.High = result.High
	tickerPrice.Low = result.Low
	tickerPrice.Last = result.Last
	tickerPrice.FirstCurrency = currency[0:3]
	tickerPrice.SecondCurrency = currency[3:]
	ticker.ProcessTicker(l.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}

func (l *LakeBTC) GetOrderbookEx(currency string) (orderbook.OrderbookBase, error) {
	ob, err := orderbook.GetOrderbook(l.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return ob, nil
	}

	var orderBook orderbook.OrderbookBase
	orderbookNew, err := l.GetOrderBook(currency)
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
	orderbook.ProcessOrderbook(l.GetName(), orderBook.FirstCurrency, orderBook.SecondCurrency, orderBook)
	return orderBook, nil
}

func (l *LakeBTC) GetExchangeAccountInfo() (exchange.ExchangeAccountInfo, error) {
	var response exchange.ExchangeAccountInfo
	response.ExchangeName = l.GetName()
	accountInfo, err := l.GetAccountInfo()
	if err != nil {
		return response, err
	}

	for x, y := range accountInfo.Balance {
		for z, w := range accountInfo.Locked {
			if z == x {
				var exchangeCurrency exchange.ExchangeAccountCurrencyInfo
				exchangeCurrency.CurrencyName = common.StringToUpper(x)
				exchangeCurrency.TotalValue, _ = strconv.ParseFloat(y, 64)
				exchangeCurrency.Hold, _ = strconv.ParseFloat(w, 64)
				response.Currencies = append(response.Currencies, exchangeCurrency)
			}
		}
	}
	return response, nil
}
