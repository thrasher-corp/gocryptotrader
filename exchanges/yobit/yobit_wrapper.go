package yobit

import (
	"errors"
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the WEX go routine
func (y *Yobit) Start() {
	go y.Run()
}

// Run implements the Yobit wrapper
func (y *Yobit) Run() {
	if y.Verbose {
		log.Printf("%s Websocket: %s.", y.GetName(), common.IsEnabled(y.Websocket))
		log.Printf("%s polling delay: %ds.\n", y.GetName(), y.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", y.GetName(), len(y.EnabledPairs), y.EnabledPairs)
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (y *Yobit) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	pairsCollated, err := exchange.GetAndFormatExchangeCurrencies(y.Name, y.GetEnabledCurrencies())
	if err != nil {
		return tickerPrice, err
	}

	result, err := y.GetTicker(pairsCollated.String())
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range y.GetEnabledCurrencies() {
		currency := exchange.FormatExchangeCurrency(y.Name, x).Lower().String()
		var tickerPrice ticker.Price
		tickerPrice.Pair = x
		tickerPrice.Last = result[currency].Last
		tickerPrice.Ask = result[currency].Sell
		tickerPrice.Bid = result[currency].Buy
		tickerPrice.Last = result[currency].Last
		tickerPrice.Low = result[currency].Low
		tickerPrice.Volume = result[currency].VolumeCurrent
		ticker.ProcessTicker(y.Name, x, tickerPrice, assetType)
	}
	return ticker.GetTicker(y.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (y *Yobit) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tick, err := ticker.GetTicker(y.GetName(), p, assetType)
	if err != nil {
		return y.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// GetOrderbookEx returns the orderbook for a currency pair
func (y *Yobit) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(y.GetName(), p, assetType)
	if err != nil {
		return y.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (y *Yobit) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := y.GetDepth(exchange.FormatExchangeCurrency(y.Name, p).String())
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

	orderbook.ProcessOrderbook(y.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(y.Name, p, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// Yobit exchange
func (y *Yobit) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = y.GetName()
	accountBalance, err := y.GetAccountInfo()
	if err != nil {
		return response, err
	}

	for x, y := range accountBalance.FundsInclOrders {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = common.StringToUpper(x)
		exchangeCurrency.TotalValue = y
		exchangeCurrency.Hold = 0
		for z, w := range accountBalance.Funds {
			if z == x {
				exchangeCurrency.Hold = y - w
			}
		}

		response.Currencies = append(response.Currencies, exchangeCurrency)
	}

	return response, nil
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (y *Yobit) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}
