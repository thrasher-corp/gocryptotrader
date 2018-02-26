package binance

import (
	"errors"
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the OKEX go routine
func (b *Binance) Start() {
	go b.Run()
}

// Run implements the OKEX wrapper
func (b *Binance) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", b.GetName(), common.IsEnabled(b.Websocket), b.WebsocketURL)
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	symbols, err := b.GetExchangeValidCurrencyPairs()
	if err != nil {
		log.Printf("%s Failed to get exchange info.\n", b.GetName())
	} else {
		forceUpgrade := false
		if !common.StringDataContains(b.EnabledPairs, "-") || !common.StringDataContains(b.AvailablePairs, "-") {
			forceUpgrade = true
		}

		if forceUpgrade {
			enabledPairs := []string{"BTC-USDT"}
			log.Println("WARNING: Available pairs for Binance reset due to config upgrade, please enable the ones you would like again")

			err = b.UpdateEnabledCurrencies(enabledPairs, true)
			if err != nil {
				log.Printf("%s Failed to get config.\n", b.GetName())
			}
		}
		err = b.UpdateAvailableCurrencies(symbols, forceUpgrade)
		if err != nil {
			log.Printf("%s Failed to get config.\n", b.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Binance) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price

	tick, err := b.GetTickers()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range b.GetEnabledCurrencies() {
		curr := exchange.FormatExchangeCurrency(b.Name, x)
		for y := range tick {
			if tick[y].Symbol == curr.String() {
				tickerPrice.Pair = x
				tickerPrice.Ask = tick[y].AskPrice
				tickerPrice.Bid = tick[y].BidPrice
				tickerPrice.High = tick[y].HighPrice
				tickerPrice.Last = tick[y].LastPrice
				tickerPrice.Low = tick[y].LowPrice
				tickerPrice.Volume = tick[y].Volume
				ticker.ProcessTicker(b.Name, x, tickerPrice, assetType)
			}
		}
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (b *Binance) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (b *Binance) GetOrderbookEx(currency pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), currency, assetType)
	if err != nil {
		return b.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Binance) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := b.GetOrderBook(exchange.FormatExchangeCurrency(b.Name, p).String(), 1000)
	if err != nil {
		return orderBook, err
	}

	for _, bids := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: bids.Quantity, Price: bids.Price})
	}

	for _, asks := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: asks.Quantity, Price: asks.Price})
	}

	orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(b.Name, p, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// Bithumb exchange
func (b *Binance) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	return response, errors.New("not implemented")
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Binance) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}
