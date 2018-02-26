package gemini

import (
	"errors"
	"log"
	"net/url"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the Gemini go routine
func (g *Gemini) Start() {
	go g.Run()
}

// Run implements the Gemini wrapper
func (g *Gemini) Run() {
	if g.Verbose {
		log.Printf("%s polling delay: %ds.\n", g.GetName(), g.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", g.GetName(), len(g.EnabledPairs), g.EnabledPairs)
	}

	exchangeProducts, err := g.GetSymbols()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", g.GetName())
	} else {
		err = g.UpdateAvailableCurrencies(exchangeProducts, false)
		if err != nil {
			log.Printf("%s Failed to get config.\n", g.GetName())
		}
	}
}

// GetExchangeAccountInfo Retrieves balances for all enabled currencies for the
// Gemini exchange
func (g *Gemini) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = g.GetName()
	accountBalance, err := g.GetBalances()
	if err != nil {
		return response, err
	}
	for i := 0; i < len(accountBalance); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = accountBalance[i].Currency
		exchangeCurrency.TotalValue = accountBalance[i].Amount
		exchangeCurrency.Hold = accountBalance[i].Available
		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (g *Gemini) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := g.GetTicker(p.Pair().String())
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Pair = p
	tickerPrice.Ask = tick.Ask
	tickerPrice.Bid = tick.Bid
	tickerPrice.Last = tick.Last
	tickerPrice.Volume = tick.Volume.USD
	ticker.ProcessTicker(g.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(g.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (g *Gemini) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(g.GetName(), p, assetType)
	if err != nil {
		return g.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (g *Gemini) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(g.GetName(), p, assetType)
	if err != nil {
		return g.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (g *Gemini) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := g.GetOrderbook(p.Pair().String(), url.Values{})
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: orderbookNew.Bids[x].Amount, Price: orderbookNew.Bids[x].Price})
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: orderbookNew.Asks[x].Amount, Price: orderbookNew.Asks[x].Price})
	}

	orderbook.ProcessOrderbook(g.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(g.Name, p, assetType)
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (g *Gemini) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}
