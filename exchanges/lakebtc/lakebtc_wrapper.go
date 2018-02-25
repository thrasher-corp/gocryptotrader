package lakebtc

import (
	"errors"
	"log"
	"strconv"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the LakeBTC go routine
func (l *LakeBTC) Start() {
	go l.Run()
}

// Run implements the LakeBTC wrapper
func (l *LakeBTC) Run() {
	if l.Verbose {
		log.Printf("%s polling delay: %ds.\n", l.GetName(), l.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", l.GetName(), len(l.EnabledPairs), l.EnabledPairs)
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (l *LakeBTC) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tick, err := l.GetTicker()
	if err != nil {
		return ticker.Price{}, err
	}

	for _, x := range l.GetEnabledCurrencies() {
		currency := exchange.FormatExchangeCurrency(l.Name, x).String()
		var tickerPrice ticker.Price
		tickerPrice.Pair = x
		tickerPrice.Ask = tick[currency].Ask
		tickerPrice.Bid = tick[currency].Bid
		tickerPrice.Volume = tick[currency].Volume
		tickerPrice.High = tick[currency].High
		tickerPrice.Low = tick[currency].Low
		tickerPrice.Last = tick[currency].Last
		ticker.ProcessTicker(l.GetName(), x, tickerPrice, assetType)
	}
	return ticker.GetTicker(l.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (l *LakeBTC) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(l.GetName(), p, assetType)
	if err != nil {
		return l.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (l *LakeBTC) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(l.GetName(), p, assetType)
	if err != nil {
		return l.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (l *LakeBTC) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := l.GetOrderBook(p.Pair().String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: orderbookNew.Bids[x].Amount, Price: orderbookNew.Bids[x].Price})
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: orderbookNew.Asks[x].Amount, Price: orderbookNew.Asks[x].Price})
	}

	orderbook.ProcessOrderbook(l.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(l.Name, p, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// LakeBTC exchange
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

// GetExchangeHistory returns historic trade data since exchange opening.
func (l *LakeBTC) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}
