package alphapoint

import (
	"log"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// GetExchangeAccountInfo retrieves balances for all enabled currencies on the
// Alphapoint exchange
func (a *Alphapoint) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = a.GetName()
	account, err := a.GetAccountInfo()
	if err != nil {
		return response, err
	}
	for i := 0; i < len(account.Currencies); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = account.Currencies[i].Name
		exchangeCurrency.TotalValue = float64(account.Currencies[i].Balance)
		exchangeCurrency.Hold = float64(account.Currencies[i].Hold)

		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	//If it all works out
	return response, nil
}

// GetTickerPrice returns the current ticker price by currency pair
func (a *Alphapoint) GetTickerPrice(p pair.CurrencyPair) ticker.TickerPrice {
	var tickerPrice ticker.TickerPrice
	tick, err := a.GetTicker(p.Pair().String())
	if err != nil {
		log.Println(err)
		return ticker.TickerPrice{}
	}
	tickerPrice.Pair = p
	tickerPrice.Ask = tick.Ask
	tickerPrice.Bid = tick.Bid
	return tickerPrice
}

// GetOrderbookEx returns an orderbookbase by currency pair
func (a *Alphapoint) GetOrderbookEx(p pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	ob, err := orderbook.GetOrderbook(a.GetName(), p)
	if err == nil {
		return ob, nil
	}

	var orderBook orderbook.OrderbookBase
	orderbookNew, err := a.GetOrderbook(p.Pair().String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.OrderbookItem{Amount: data.Quantity, Price: data.Price})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.OrderbookItem{Amount: data.Quantity, Price: data.Price})
	}

	orderBook.Pair = p
	orderbook.ProcessOrderbook(a.GetName(), p, orderBook)
	return orderBook, nil
}
