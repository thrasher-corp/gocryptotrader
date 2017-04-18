package okcoin

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (o *OKCoin) Start() {
	go o.Run()
}

func (o *OKCoin) Run() {
	if o.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", o.GetName(), common.IsEnabled(o.Websocket), o.WebsocketURL)
		log.Printf("%s polling delay: %ds.\n", o.GetName(), o.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", o.GetName(), len(o.EnabledPairs), o.EnabledPairs)
	}

	if o.Websocket {
		go o.WebsocketClient()
	}

	for o.Enabled {
		for _, x := range o.EnabledPairs {
			curr := pair.NewCurrencyPair(x[0:3], x[3:])
			curr.Delimiter = "_"
			if o.APIUrl == OKCOIN_API_URL {
				for _, y := range o.FuturesValues {
					futuresValue := y
					go func() {
						ticker, err := o.GetFuturesTicker(curr.Pair().Lower().String(), futuresValue)
						if err != nil {
							log.Println(err)
							return
						}
						log.Printf("OKCoin Intl Futures %s (%s): Last %f High %f Low %f Volume %f\n", curr.Pair().String(), futuresValue, ticker.Last, ticker.High, ticker.Low, ticker.Vol)
						stats.AddExchangeInfo(o.GetName(), curr.GetFirstCurrency().String(), curr.GetSecondCurrency().String(), ticker.Last, ticker.Vol)
					}()
				}
				go func() {
					ticker, err := o.GetTickerPrice(curr)
					if err != nil {
						log.Println(err)
						return
					}
					log.Printf("OKCoin Intl Spot %s: Last %f High %f Low %f Volume %f\n", curr.Pair().String(), ticker.Last, ticker.High, ticker.Low, ticker.Volume)
					stats.AddExchangeInfo(o.GetName(), curr.GetFirstCurrency().String(), curr.GetSecondCurrency().String(), ticker.Last, ticker.Volume)
				}()
			} else {
				go func() {
					ticker, err := o.GetTickerPrice(curr)
					if err != nil {
						log.Println(err)
						return
					}
					tickerLastUSD, _ := currency.ConvertCurrency(ticker.Last, "CNY", "USD")
					tickerHighUSD, _ := currency.ConvertCurrency(ticker.High, "CNY", "USD")
					tickerLowUSD, _ := currency.ConvertCurrency(ticker.Low, "CNY", "USD")
					log.Printf("OKCoin China %s: Last %f (%f) High %f (%f) Low %f (%f) Volume %f\n", curr.Pair().String(), tickerLastUSD, ticker.Last, tickerHighUSD, ticker.High, tickerLowUSD, ticker.Low, ticker.Volume)
					stats.AddExchangeInfo(o.GetName(), curr.GetFirstCurrency().String(), curr.GetSecondCurrency().String(), ticker.Last, ticker.Volume)
					stats.AddExchangeInfo(o.GetName(), curr.GetFirstCurrency().String(), "USD", tickerLastUSD, ticker.Volume)
				}()
			}
		}
		time.Sleep(time.Second * o.RESTPollingDelay)
	}
}

func (o *OKCoin) GetTickerPrice(currency pair.CurrencyPair) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(o.GetName(), currency)
	if err == nil {
		return tickerNew, nil
	}

	var tickerPrice ticker.TickerPrice
	tick, err := o.GetTicker(currency.Pair().Lower().String())
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Pair = currency
	tickerPrice.Ask = tick.Sell
	tickerPrice.Bid = tick.Buy
	tickerPrice.Low = tick.Low
	tickerPrice.Last = tick.Last
	tickerPrice.Volume = tick.Vol
	tickerPrice.High = tick.High
	ticker.ProcessTicker(o.GetName(), currency, tickerPrice)
	return tickerPrice, nil
}

func (o *OKCoin) GetOrderbookEx(currency pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	ob, err := orderbook.GetOrderbook(o.GetName(), currency)
	if err == nil {
		return ob, nil
	}

	var orderBook orderbook.OrderbookBase
	orderbookNew, err := o.GetOrderBook(currency.Pair().Lower().String(), 200, false)
	if err != nil {
		return orderBook, err
	}

	for x, _ := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.OrderbookItem{Amount: data[1], Price: data[0]})
	}

	for x, _ := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.OrderbookItem{Amount: data[1], Price: data[0]})
	}
	orderBook.Pair = currency
	orderbook.ProcessOrderbook(o.GetName(), currency, orderBook)
	return orderBook, nil
}

func (e *OKCoin) GetExchangeAccountInfo() (exchange.ExchangeAccountInfo, error) {
	var response exchange.ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	assets, err := e.GetUserInfo()
	if err != nil {
		return response, err
	}

	response.Currencies = append(response.Currencies, exchange.ExchangeAccountCurrencyInfo{
		CurrencyName: "BTC",
		TotalValue:   assets.Info.Funds.Free.BTC,
		Hold:         assets.Info.Funds.Freezed.BTC,
	})

	response.Currencies = append(response.Currencies, exchange.ExchangeAccountCurrencyInfo{
		CurrencyName: "LTC",
		TotalValue:   assets.Info.Funds.Free.LTC,
		Hold:         assets.Info.Funds.Freezed.LTC,
	})

	response.Currencies = append(response.Currencies, exchange.ExchangeAccountCurrencyInfo{
		CurrencyName: "USD",
		TotalValue:   assets.Info.Funds.Free.USD,
		Hold:         assets.Info.Funds.Freezed.USD,
	})

	response.Currencies = append(response.Currencies, exchange.ExchangeAccountCurrencyInfo{
		CurrencyName: "CNY",
		TotalValue:   assets.Info.Funds.Free.CNY,
		Hold:         assets.Info.Funds.Freezed.CNY,
	})

	return response, nil
}
