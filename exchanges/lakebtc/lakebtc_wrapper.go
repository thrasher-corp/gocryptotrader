package lakebtc

import (
	"log"
	"strconv"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
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
		pairs := l.GetEnabledCurrencies()
		for x := range pairs {
			currency := pairs[x]
			ticker, err := l.UpdateTicker(currency)
			if err != nil {
				log.Println(err)
				continue
			}
			log.Printf("LakeBTC %s: Last %f High %f Low %f Volume %f\n", exchange.FormatCurrency(currency).String(), ticker.Last, ticker.High, ticker.Low, ticker.Volume)
			stats.AddExchangeInfo(l.GetName(), currency.GetFirstCurrency().String(), currency.GetSecondCurrency().String(), ticker.Last, ticker.Volume)
		}
		time.Sleep(time.Second * l.RESTPollingDelay)
	}
}

func (l *LakeBTC) UpdateTicker(p pair.CurrencyPair) (ticker.TickerPrice, error) {
	tick, err := l.GetTicker()
	if err != nil {
		return ticker.TickerPrice{}, err
	}

	result, ok := tick[p.Pair().String()]
	if !ok {
		return ticker.TickerPrice{}, err
	}

	var tickerPrice ticker.TickerPrice
	tickerPrice.Pair = p
	tickerPrice.Ask = result.Ask
	tickerPrice.Bid = result.Bid
	tickerPrice.Volume = result.Volume
	tickerPrice.High = result.High
	tickerPrice.Low = result.Low
	tickerPrice.Last = result.Last
	ticker.ProcessTicker(l.GetName(), p, tickerPrice)
	return tickerPrice, nil
}

func (l *LakeBTC) GetTickerPrice(p pair.CurrencyPair) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(l.GetName(), p)
	if err != nil {
		return l.UpdateTicker(p)
	}
	return tickerNew, nil
}

func (l *LakeBTC) GetOrderbookEx(p pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	ob, err := orderbook.GetOrderbook(l.GetName(), p)
	if err == nil {
		return ob, nil
	}

	var orderBook orderbook.OrderbookBase
	orderbookNew, err := l.GetOrderBook(p.Pair().String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.OrderbookItem{Amount: orderbookNew.Bids[x].Amount, Price: orderbookNew.Bids[x].Price})
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.OrderbookItem{Amount: orderbookNew.Asks[x].Amount, Price: orderbookNew.Asks[x].Price})
	}

	orderBook.Pair = p
	orderbook.ProcessOrderbook(l.GetName(), p, orderBook)
	return orderBook, nil
}

func (l *LakeBTC) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = l.GetName()
	accountInfo, err := l.GetAccountInfo()
	if err != nil {
		return response, err
	}

	for x, y := range accountInfo.Balance {
		for z, w := range accountInfo.Locked {
			if z == x {
				var exchangeCurrency exchange.AccountCurrencyInfo
				exchangeCurrency.CurrencyName = common.StringToUpper(x)
				exchangeCurrency.TotalValue, _ = strconv.ParseFloat(y, 64)
				exchangeCurrency.Hold, _ = strconv.ParseFloat(w, 64)
				response.Currencies = append(response.Currencies, exchangeCurrency)
			}
		}
	}
	return response, nil
}
