package poloniex

import (
	"errors"
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the Poloniex go routine
func (po *Poloniex) Start() {
	go po.Run()
}

// Run implements the Poloniex wrapper
func (po *Poloniex) Run() {
	if po.Verbose {
		log.Printf("%s Websocket: %s (url: %s).\n", po.GetName(), common.IsEnabled(po.Websocket), poloniexWebsocketAddress)
		log.Printf("%s polling delay: %ds.\n", po.GetName(), po.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", po.GetName(), len(po.EnabledPairs), po.EnabledPairs)
	}

	if po.Websocket {
		go po.WebsocketClient()
	}

	exchangeCurrencies, err := po.GetExchangeCurrencies()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", po.GetName())
	} else {
		forceUpdate := false
		if common.StringDataCompare(po.AvailablePairs, "BTC_USDT") {
			log.Printf("%s contains invalid pair, forcing upgrade of available currencies.\n",
				po.GetName())
			forceUpdate = true
		}
		err = po.UpdateAvailableCurrencies(exchangeCurrencies, forceUpdate)
		if err != nil {
			log.Printf("%s Failed to update available currencies %s.\n", po.GetName(), err)
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (po *Poloniex) UpdateTicker(currencyPair pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := po.GetTicker()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range po.GetEnabledCurrencies() {
		var tp ticker.Price
		curr := exchange.FormatExchangeCurrency(po.GetName(), x).String()
		tp.Pair = x
		tp.Ask = tick[curr].LowestAsk
		tp.Bid = tick[curr].HighestBid
		tp.High = tick[curr].High24Hr
		tp.Last = tick[curr].Last
		tp.Low = tick[curr].Low24Hr
		tp.Volume = tick[curr].BaseVolume
		ticker.ProcessTicker(po.GetName(), x, tp, assetType)
	}
	return ticker.GetTicker(po.Name, currencyPair, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (po *Poloniex) GetTickerPrice(currencyPair pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(po.GetName(), currencyPair, assetType)
	if err != nil {
		return po.UpdateTicker(currencyPair, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (po *Poloniex) GetOrderbookEx(currencyPair pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(po.GetName(), currencyPair, assetType)
	if err != nil {
		return po.UpdateOrderbook(currencyPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (po *Poloniex) UpdateOrderbook(currencyPair pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := po.GetOrderbook("", 1000)
	if err != nil {
		return orderBook, err
	}

	for _, x := range po.GetEnabledCurrencies() {
		currency := exchange.FormatExchangeCurrency(po.Name, x).String()
		data, ok := orderbookNew.Data[currency]
		if !ok {
			continue
		}
		orderBook.Pair = x

		var obItems []orderbook.Item
		for y := range data.Bids {
			obData := data.Bids[y]
			obItems = append(obItems, orderbook.Item{Amount: obData.Amount, Price: obData.Price})
		}

		orderBook.Bids = obItems
		obItems = []orderbook.Item{}
		for y := range data.Asks {
			obData := data.Asks[y]
			obItems = append(obItems, orderbook.Item{Amount: obData.Amount, Price: obData.Price})
		}
		orderBook.Asks = obItems
		orderbook.ProcessOrderbook(po.Name, x, orderBook, assetType)
	}
	return orderbook.GetOrderbook(po.Name, currencyPair, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// Poloniex exchange
func (po *Poloniex) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = po.GetName()
	accountBalance, err := po.GetBalances()
	if err != nil {
		return response, err
	}

	for x, y := range accountBalance.Currency {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = x
		exchangeCurrency.TotalValue = y
		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (po *Poloniex) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}
