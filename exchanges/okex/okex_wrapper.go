package okex

import (
	"errors"
	"log"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the OKEX go routine
func (o *OKEX) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		o.Run()
		wg.Done()
	}()
}

// Run implements the OKEX wrapper
func (o *OKEX) Run() {
	if o.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", o.GetName(), common.IsEnabled(o.Websocket), o.WebsocketURL)
		log.Printf("%s polling delay: %ds.\n", o.GetName(), o.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", o.GetName(), len(o.EnabledPairs), o.EnabledPairs)
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (o *OKEX) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	currency := exchange.FormatExchangeCurrency(o.Name, p).String()
	var tickerPrice ticker.Price

	if assetType != ticker.Spot {
		if p.SecondCurrency.String() == common.StringToLower("USDT") {
			p.SecondCurrency = "usd"
			currency = exchange.FormatExchangeCurrency(o.Name, p).String()
		}
		tick, err := o.GetContractPrice(currency, assetType)
		if err != nil {
			return tickerPrice, err
		}

		tickerPrice.Pair = p
		tickerPrice.Ask = tick.Ticker.Sell
		tickerPrice.Bid = tick.Ticker.Buy
		tickerPrice.Low = tick.Ticker.Low
		tickerPrice.Last = tick.Ticker.Last
		tickerPrice.Volume = tick.Ticker.Vol
		tickerPrice.High = tick.Ticker.High
		ticker.ProcessTicker(o.GetName(), p, tickerPrice, assetType)
	} else {
		if p.SecondCurrency.String() == common.StringToLower("USD") {
			p.SecondCurrency = "usdt"
			currency = exchange.FormatExchangeCurrency(o.Name, p).String()
		}

		tick, err := o.GetSpotTicker(currency)
		if err != nil {
			return tickerPrice, err
		}
		tickerPrice.Pair = p
		tickerPrice.Ask = tick.Ticker.Sell
		tickerPrice.Bid = tick.Ticker.Buy
		tickerPrice.Low = tick.Ticker.Low
		tickerPrice.Last = tick.Ticker.Last
		tickerPrice.Volume = tick.Ticker.Vol
		tickerPrice.High = tick.Ticker.High
		ticker.ProcessTicker(o.GetName(), p, tickerPrice, ticker.Spot)

	}
	return ticker.GetTicker(o.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (o *OKEX) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(o.GetName(), p, assetType)
	if err != nil {
		return o.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (o *OKEX) GetOrderbookEx(currency pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(o.GetName(), currency, assetType)
	if err != nil {
		return o.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (o *OKEX) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	currency := exchange.FormatExchangeCurrency(o.Name, p).String()

	if assetType != ticker.Spot {
		if p.SecondCurrency.String() == common.StringToLower("USDT") {
			p.SecondCurrency = "usd"
			currency = exchange.FormatExchangeCurrency(o.Name, p).String()
		}
		orderbookNew, err := o.GetContractMarketDepth(currency, assetType)
		if err != nil {
			return orderBook, err
		}

		for x := range orderbookNew.Bids {
			data := orderbookNew.Bids[x]
			orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data.Volume, Price: data.Price})
		}

		for x := range orderbookNew.Asks {
			data := orderbookNew.Asks[x]
			orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data.Volume, Price: data.Price})
		}

	} else {
		if p.SecondCurrency.String() == common.StringToLower("USD") {
			p.SecondCurrency = "usdt"
			currency = exchange.FormatExchangeCurrency(o.Name, p).String()
		}

		orderbookNew, err := o.GetSpotMarketDepth(currency, "200")
		if err != nil {
			return orderBook, err
		}

		for x := range orderbookNew.Bids {
			data := orderbookNew.Bids[x]
			orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data.Volume, Price: data.Price})
		}

		for x := range orderbookNew.Asks {
			data := orderbookNew.Asks[x]
			orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data.Volume, Price: data.Price})
		}
	}

	orderbook.ProcessOrderbook(o.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(o.Name, p, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// OKEX exchange
func (o *OKEX) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	return response, errors.New("not implemented")
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (o *OKEX) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}

// SubmitExchangeOrder submits a new order
func (o *OKEX) SubmitExchangeOrder(p pair.CurrencyPair, side string, orderType int, amount, price float64) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// ModifyExchangeOrder will allow of changing orderbook placement and limit to
// market conversion
func (o *OKEX) ModifyExchangeOrder(p pair.CurrencyPair, orderID, action int64) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelExchangeOrder cancels an order by its corresponding ID number
func (o *OKEX) CancelExchangeOrder(p pair.CurrencyPair, orderID int64) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelAllExchangeOrders cancels all orders associated with a currency pair
func (o *OKEX) CancelAllExchangeOrders(p pair.CurrencyPair) error {
	return errors.New("not yet implemented")
}

// GetExchangeOrderInfo returns information on a current open order
func (o *OKEX) GetExchangeOrderInfo(orderID int64) (float64, error) {
	return 0, errors.New("not yet implemented")
}

// GetExchangeDepositAddress returns a deposit address for a specified currency
func (o *OKEX) GetExchangeDepositAddress(p pair.CurrencyPair) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawExchangeFunds returns a withdrawal ID when a withdrawal is submitted
func (o *OKEX) WithdrawExchangeFunds(address string, p pair.CurrencyPair, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}
