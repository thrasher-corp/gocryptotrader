package bithumb

import (
	"errors"
	"log"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the OKEX go routine
func (b *Bithumb) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the OKEX wrapper
func (b *Bithumb) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", b.GetName(), common.IsEnabled(b.Websocket), b.WebsocketURL)
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bithumb) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	item := p.GetFirstCurrency().String()
	tick, err := b.GetTicker(item)
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = p
	tickerPrice.Ask = tick.Data.SellPrice
	tickerPrice.Bid = tick.Data.BuyPrice
	tickerPrice.Low = tick.Data.MinPrice
	tickerPrice.Last = tick.Data.ClosingPrice
	tickerPrice.Volume = tick.Data.Volume1Day
	tickerPrice.High = tick.Data.MaxPrice
	ticker.ProcessTicker(b.GetName(), p, tickerPrice, assetType)

	return ticker.GetTicker(b.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (b *Bithumb) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (b *Bithumb) GetOrderbookEx(currency pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), currency, assetType)
	if err != nil {
		return b.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bithumb) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	currency := p.GetFirstCurrency().String()

	orderbookNew, err := b.GetOrderBook(currency)
	if err != nil {
		return orderBook, err
	}

	for _, bids := range orderbookNew.Data.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: bids.Quantity, Price: bids.Price})
	}

	for _, asks := range orderbookNew.Data.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: asks.Quantity, Price: asks.Price})
	}

	orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(b.Name, p, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// Bithumb exchange
func (b *Bithumb) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	return response, errors.New("not implemented")
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Bithumb) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}

// SubmitExchangeOrder submits a new order
func (b *Bithumb) SubmitExchangeOrder(p pair.CurrencyPair, side string, orderType int, amount, price float64) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// ModifyExchangeOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bithumb) ModifyExchangeOrder(p pair.CurrencyPair, orderID, action int64) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelExchangeOrder cancels an order by its corresponding ID number
func (b *Bithumb) CancelExchangeOrder(p pair.CurrencyPair, orderID int64) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelAllExchangeOrders cancels all orders associated with a currency pair
func (b *Bithumb) CancelAllExchangeOrders(p pair.CurrencyPair) error {
	return errors.New("not yet implemented")
}

// GetExchangeOrderInfo returns information on a current open order
func (b *Bithumb) GetExchangeOrderInfo(orderID int64) (float64, error) {
	return 0, errors.New("not yet implemented")
}

// GetExchangeDepositAddress returns a deposit address for a specified currency
func (b *Bithumb) GetExchangeDepositAddress(p pair.CurrencyPair) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawExchangeFunds returns a withdrawal ID when a withdrawal is submitted
func (b *Bithumb) WithdrawExchangeFunds(address string, p pair.CurrencyPair, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}
