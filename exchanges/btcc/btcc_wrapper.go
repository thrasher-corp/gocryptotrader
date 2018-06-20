package btcc

import (
	"errors"
	"log"
	"sync"

	"github.com/idoall/gocryptotrader/common"
	"github.com/idoall/gocryptotrader/config"
	"github.com/idoall/gocryptotrader/currency/pair"
	exchange "github.com/idoall/gocryptotrader/exchanges"
	"github.com/idoall/gocryptotrader/exchanges/orderbook"
	"github.com/idoall/gocryptotrader/exchanges/ticker"
)

// Start starts the BTCC go routine
func (b *BTCC) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the BTCC wrapper
func (b *BTCC) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s.", b.GetName(), common.IsEnabled(b.Websocket))
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	if b.Websocket {
		go b.WebsocketClient()
	}

	if common.StringDataContains(b.EnabledPairs, "CNY") || common.StringDataContains(b.AvailablePairs, "CNY") || common.StringDataContains(b.BaseCurrencies, "CNY") {
		log.Println("WARNING: BTCC only supports BTCUSD now, upgrading available, enabled and base currencies to BTCUSD/USD")
		pairs := []string{"BTCUSD"}
		cfg := config.GetConfig()
		exchCfg, err := cfg.GetExchangeConfig(b.Name)
		if err != nil {
			log.Printf("%s failed to get exchange config. %s\n", b.Name, err)
			return
		}

		exchCfg.BaseCurrencies = "USD"
		exchCfg.AvailablePairs = pairs[0]
		exchCfg.EnabledPairs = pairs[0]
		b.BaseCurrencies = []string{"USD"}

		err = b.UpdateCurrencies(pairs, false, true)
		if err != nil {
			log.Printf("%s failed to update available currencies. %s\n", b.Name, err)
		}

		err = b.UpdateCurrencies(pairs, true, true)
		if err != nil {
			log.Printf("%s failed to update enabled currencies. %s\n", b.Name, err)
		}

		err = cfg.UpdateExchangeConfig(exchCfg)
		if err != nil {
			log.Printf("%s failed to update config. %s\n", b.Name, err)
			return
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *BTCC) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := b.GetTicker(exchange.FormatExchangeCurrency(b.GetName(), p).String())
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Pair = p
	tickerPrice.Ask = tick.AskPrice
	tickerPrice.Bid = tick.BidPrice
	tickerPrice.Low = tick.Low
	tickerPrice.Last = tick.Last
	tickerPrice.Volume = tick.Volume24H
	tickerPrice.High = tick.High
	ticker.ProcessTicker(b.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(b.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (b *BTCC) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns the orderbook for a currency pair
func (b *BTCC) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTCC) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := b.GetOrderBook(exchange.FormatExchangeCurrency(b.GetName(), p).String(), 100)
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Price: data[0], Amount: data[1]})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Price: data[0], Amount: data[1]})
	}

	orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(b.Name, p, assetType)
}

// GetExchangeAccountInfo : Retrieves balances for all enabled currencies for
// the Kraken exchange - TODO
func (b *BTCC) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = b.GetName()
	return response, nil
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *BTCC) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}

// SubmitExchangeOrder submits a new order
func (b *BTCC) SubmitExchangeOrder(p pair.CurrencyPair, side string, orderType int, amount, price float64) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// ModifyExchangeOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *BTCC) ModifyExchangeOrder(p pair.CurrencyPair, orderID, action int64) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelExchangeOrder cancels an order by its corresponding ID number
func (b *BTCC) CancelExchangeOrder(p pair.CurrencyPair, orderID int64) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelAllExchangeOrders cancels all orders associated with a currency pair
func (b *BTCC) CancelAllExchangeOrders(p pair.CurrencyPair) error {
	return errors.New("not yet implemented")
}

// GetExchangeOrderInfo returns information on a current open order
func (b *BTCC) GetExchangeOrderInfo(orderID int64) (float64, error) {
	return 0, errors.New("not yet implemented")
}

// GetExchangeDepositAddress returns a deposit address for a specified currency
func (b *BTCC) GetExchangeDepositAddress(p pair.CurrencyPair) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawExchangeFunds returns a withdrawal ID when a withdrawal is submitted
func (b *BTCC) WithdrawExchangeFunds(address string, p pair.CurrencyPair, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}
