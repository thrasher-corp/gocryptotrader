package okcoin

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

// Start starts the OKCoin go routine
func (o *OKCoin) Start() {
	go o.Run()
}

// Run implements the OKCoin wrapper
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
		pairs := o.GetEnabledCurrencies()
		for x := range pairs {
			curr := pairs[x]
			if o.APIUrl == OKCOIN_API_URL {
				for _, y := range o.FuturesValues {
					futuresValue := y
					go func() {
						ticker, err := o.GetFuturesTicker(exchange.FormatExchangeCurrency(o.Name, curr).String(), futuresValue)
						if err != nil {
							log.Println(err)
							return
						}
						log.Printf("OKCoin Intl Futures %s (%s): Last %f High %f Low %f Volume %f\n", exchange.FormatCurrency(curr).String(), futuresValue, ticker.Last, ticker.High, ticker.Low, ticker.Vol)
						stats.AddExchangeInfo(o.GetName(), curr.GetFirstCurrency().String(), curr.GetSecondCurrency().String(), ticker.Last, ticker.Vol)
					}()
				}
			}
		}
		time.Sleep(time.Second * o.RESTPollingDelay)
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (o *OKCoin) UpdateTicker(currency pair.CurrencyPair) (ticker.TickerPrice, error) {
	var tickerPrice ticker.TickerPrice
	tick, err := o.GetTicker(exchange.FormatExchangeCurrency(o.Name, currency).String())
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

// GetTickerPrice returns the ticker for a currency pair
func (o *OKCoin) GetTickerPrice(currency pair.CurrencyPair) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(o.GetName(), currency)
	if err != nil {
		return o.UpdateTicker(currency)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (o *OKCoin) GetOrderbookEx(currency pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	ob, err := orderbook.GetOrderbook(o.GetName(), currency)
	if err == nil {
		return o.UpdateOrderbook(currency)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (o *OKCoin) UpdateOrderbook(currency pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	var orderBook orderbook.OrderbookBase
	orderbookNew, err := o.GetOrderBook(exchange.FormatExchangeCurrency(o.Name, currency).String(), 200, false)
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.OrderbookItem{Amount: data[1], Price: data[0]})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.OrderbookItem{Amount: data[1], Price: data[0]})
	}

	orderBook.Pair = currency
	orderbook.ProcessOrderbook(o.GetName(), currency, orderBook)
	return orderBook, nil
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// OKCoin exchange
func (o *OKCoin) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = o.GetName()
	assets, err := o.GetUserInfo()
	if err != nil {
		return response, err
	}

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "BTC",
		TotalValue:   assets.Info.Funds.Free.BTC,
		Hold:         assets.Info.Funds.Freezed.BTC,
	})

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "LTC",
		TotalValue:   assets.Info.Funds.Free.LTC,
		Hold:         assets.Info.Funds.Freezed.LTC,
	})

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "USD",
		TotalValue:   assets.Info.Funds.Free.USD,
		Hold:         assets.Info.Funds.Freezed.USD,
	})

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "CNY",
		TotalValue:   assets.Info.Funds.Free.CNY,
		Hold:         assets.Info.Funds.Freezed.CNY,
	})

	return response, nil
}
