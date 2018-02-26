package alphapoint

import (
	"errors"

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

// UpdateTicker updates and returns the ticker for a currency pair
func (a *Alphapoint) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := a.GetTicker(p.Pair().String())
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = p
	tickerPrice.Ask = tick.Ask
	tickerPrice.Bid = tick.Bid
	tickerPrice.Low = tick.Low
	tickerPrice.High = tick.High
	tickerPrice.Volume = tick.Volume
	tickerPrice.Last = tick.Last
	ticker.ProcessTicker(a.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(a.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (a *Alphapoint) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tick, err := ticker.GetTicker(a.GetName(), p, assetType)
	if err != nil {
		return a.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (a *Alphapoint) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := a.GetOrderbook(p.Pair().String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data.Quantity, Price: data.Price})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data.Quantity, Price: data.Price})
	}

	orderbook.ProcessOrderbook(a.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(a.Name, p, assetType)
}

// GetOrderbookEx returns the orderbook for a currency pair
func (a *Alphapoint) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(a.GetName(), p, assetType)
	if err != nil {
		return a.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (a *Alphapoint) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}
