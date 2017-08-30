package liqui

import (
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the Liqui go routine
func (l *Liqui) Start() {
	go l.Run()
}

// Run implements the Liqui wrapper
func (l *Liqui) Run() {
	if l.Verbose {
		log.Printf("%s polling delay: %ds.\n", l.GetName(), l.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", l.GetName(), len(l.EnabledPairs), l.EnabledPairs)
	}

	var err error
	l.Info, err = l.GetInfo()
	if err != nil {
		log.Printf("%s Unable to fetch info.\n", l.GetName())
	} else {
		exchangeProducts := l.GetAvailablePairs(true)
		err = l.UpdateAvailableCurrencies(exchangeProducts, false)
		if err != nil {
			log.Printf("%s Failed to get config.\n", l.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (l *Liqui) UpdateTicker(p pair.CurrencyPair) (ticker.TickerPrice, error) {
	var tickerPrice ticker.TickerPrice
	pairsString, err := exchange.GetAndFormatExchangeCurrencies(l.Name,
		l.GetEnabledCurrencies())
	if err != nil {
		return tickerPrice, err
	}

	result, err := l.GetTicker(pairsString.String())
	if err != nil {
		return tickerPrice, err
	}

	for x, y := range result {
		var tp ticker.TickerPrice
		currency := pair.NewCurrencyPairDelimiter(common.StringToUpper(x), "_")
		tp.Pair = currency
		tp.Last = y.Last
		tp.Ask = y.Sell
		tp.Bid = y.Buy
		tp.Last = y.Last
		tp.Low = y.Low
		tp.Volume = y.Vol_cur
		ticker.ProcessTicker(l.GetName(), currency, tp)
	}

	return ticker.GetTicker(l.GetName(), p)
}

// GetTickerPrice returns the ticker for a currency pair
func (l *Liqui) GetTickerPrice(p pair.CurrencyPair) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(l.GetName(), p)
	if err != nil {
		return l.UpdateTicker(p)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (l *Liqui) GetOrderbookEx(p pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	ob, err := orderbook.GetOrderbook(l.GetName(), p)
	if err == nil {
		return l.UpdateOrderbook(p)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (l *Liqui) UpdateOrderbook(p pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	var orderBook orderbook.OrderbookBase
	orderbookNew, err := l.GetDepth(exchange.FormatExchangeCurrency(l.Name, p).String())
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

	orderBook.Pair = p
	orderbook.ProcessOrderbook(l.GetName(), p, orderBook)
	return orderBook, nil
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// Liqui exchange
func (l *Liqui) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = l.GetName()
	accountBalance, err := l.GetAccountInfo()
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
