package anx

import (
	"errors"
	"log"
	"strconv"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the ANX go routine
func (a *ANX) Start() {
	go a.Run()
}

// Run implements the ANX wrapper
func (a *ANX) Run() {
	if a.Verbose {
		log.Printf("%s polling delay: %ds.\n", a.GetName(), a.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", a.GetName(), len(a.EnabledPairs), a.EnabledPairs)
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (a *ANX) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := a.GetTicker(exchange.FormatExchangeCurrency(a.GetName(), p).String())
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = p

	if tick.Data.Sell.Value != "" {
		tickerPrice.Ask, err = strconv.ParseFloat(tick.Data.Sell.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Ask = 0
	}

	if tick.Data.Buy.Value != "" {
		tickerPrice.Bid, err = strconv.ParseFloat(tick.Data.Buy.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Bid = 0
	}

	if tick.Data.Low.Value != "" {
		tickerPrice.Low, err = strconv.ParseFloat(tick.Data.Low.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Low = 0
	}

	if tick.Data.Last.Value != "" {
		tickerPrice.Last, err = strconv.ParseFloat(tick.Data.Last.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Last = 0
	}

	if tick.Data.Vol.Value != "" {
		tickerPrice.Volume, err = strconv.ParseFloat(tick.Data.Vol.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Volume = 0
	}

	if tick.Data.High.Value != "" {
		tickerPrice.High, err = strconv.ParseFloat(tick.Data.High.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.High = 0
	}
	ticker.ProcessTicker(a.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(a.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (a *ANX) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(a.GetName(), p, assetType)
	if err != nil {
		return a.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns the orderbook for a currency pair
func (a *ANX) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(a.GetName(), p, assetType)
	if err != nil {
		return a.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (a *ANX) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := a.GetDepth(exchange.FormatExchangeCurrency(a.GetName(), p).String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Data.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Price: orderbookNew.Data.Asks[x].Price, Amount: orderbookNew.Data.Asks[x].Amount})
	}

	for x := range orderbookNew.Data.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Price: orderbookNew.Data.Bids[x].Price, Amount: orderbookNew.Data.Bids[x].Amount})
	}

	orderbook.ProcessOrderbook(a.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(a.Name, p, assetType)
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the ANX exchange
func (a *ANX) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = a.GetName()
	return response, nil
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (a *ANX) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}
