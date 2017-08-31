package btce

import (
	"errors"
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the BTCE go routine
func (b *BTCE) Start() {
	go b.Run()
}

// Run implements the BTCE wrapper
func (b *BTCE) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s.", b.GetName(), common.IsEnabled(b.Websocket))
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	pairs := b.GetEnabledCurrencies()
	pairsCollated, err := exchange.GetAndFormatExchangeCurrencies(b.Name, pairs)
	if err != nil {
		log.Println(err)
		b.Enabled = false
		return
	}

	for b.Enabled {
		go func() {
			ticker, err := b.GetTicker(pairsCollated.String())
			if err != nil {
				log.Println(err)
				return
			}
			for x, y := range ticker {
				x = common.StringToUpper(x[0:3] + x[4:])
				log.Printf("BTC-e %s: Last %f High %f Low %f Volume %f\n", x, y.Last, y.High, y.Low, y.Vol_cur)
				b.Ticker[x] = y
				stats.AddExchangeInfo(b.GetName(), common.StringToUpper(x[0:3]), common.StringToUpper(x[4:]), y.Last, y.Vol_cur)
			}
		}()
		time.Sleep(time.Second * b.RESTPollingDelay)
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *BTCE) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	result, err := b.GetTicker(p.Pair().String())
	if err != nil {
		return tickerPrice, err
	}

	tick, ok := result[p.Pair().Lower().String()]
	if !ok {
		return tickerPrice, errors.New("unable to get currency")
	}
	tickerPrice.Pair = p
	tickerPrice.Ask = tick.Buy
	tickerPrice.Bid = tick.Sell
	tickerPrice.Low = tick.Low
	tickerPrice.Last = tick.Last
	tickerPrice.Volume = tick.Vol_cur
	tickerPrice.High = tick.High
	ticker.ProcessTicker(b.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(b.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (b *BTCE) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tick, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// GetOrderbookEx returns the orderbook for a currency pair
func (b *BTCE) GetOrderbookEx(p pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), p)
	if err == nil {
		return b.UpdateOrderbook(p)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTCE) UpdateOrderbook(p pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	var orderBook orderbook.OrderbookBase
	orderbookNew, err := b.GetDepth(exchange.FormatExchangeCurrency(b.Name, p).String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.OrderbookItem{Price: data[0], Amount: data[1]})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.OrderbookItem{Price: data[0], Amount: data[1]})
	}

	orderbook.ProcessOrderbook(b.GetName(), p, orderBook)
	return orderbook.GetOrderbook(b.Name, p)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// BTCE exchange
func (b *BTCE) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = b.GetName()
	accountBalance, err := b.GetAccountInfo()
	if err != nil {
		return response, err
	}

	for x, y := range accountBalance.Funds {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = common.StringToUpper(x)
		exchangeCurrency.TotalValue = y
		exchangeCurrency.Hold = 0
		response.Currencies = append(response.Currencies, exchangeCurrency)
	}

	return response, nil
}
