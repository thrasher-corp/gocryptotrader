package bitstamp

import (
	"errors"
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the Bitstamp go routine
func (b *Bitstamp) Start() {
	go b.Run()
}

// Run implements the Bitstamp wrapper
func (b *Bitstamp) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s.", b.GetName(), common.IsEnabled(b.Websocket))
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	if b.Websocket {
		go b.PusherClient()
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitstamp) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := b.GetTicker(p.Pair().String(), false)
	if err != nil {
		return tickerPrice, err

	}
	tickerPrice.Pair = p
	tickerPrice.Ask = tick.Ask
	tickerPrice.Bid = tick.Bid
	tickerPrice.Low = tick.Low
	tickerPrice.Last = tick.Last
	tickerPrice.Volume = tick.Volume
	tickerPrice.High = tick.High
	ticker.ProcessTicker(b.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(b.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (b *Bitstamp) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tick, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// GetOrderbookEx returns the orderbook for a currency pair
func (b *Bitstamp) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitstamp) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := b.GetOrderbook(p.Pair().String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data.Amount, Price: data.Price})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data.Amount, Price: data.Price})
	}

	orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(b.Name, p, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// Bitstamp exchange
func (b *Bitstamp) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = b.GetName()
	accountBalance, err := b.GetBalance()
	if err != nil {
		return response, err
	}

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "BTC",
		TotalValue:   accountBalance.BTCAvailable,
		Hold:         accountBalance.BTCReserved,
	})

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "XRP",
		TotalValue:   accountBalance.XRPAvailable,
		Hold:         accountBalance.XRPReserved,
	})

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "USD",
		TotalValue:   accountBalance.USDAvailable,
		Hold:         accountBalance.USDReserved,
	})

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "EUR",
		TotalValue:   accountBalance.EURAvailable,
		Hold:         accountBalance.EURReserved,
	})
	return response, nil
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Bitstamp) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}
