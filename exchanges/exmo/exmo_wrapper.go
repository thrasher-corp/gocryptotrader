package exmo

import (
	"log"
	"strconv"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the EXMO go routine
func (e *EXMO) Start() {
	go e.Run()
}

// Run implements the EXMO wrapper
func (e *EXMO) Run() {
	if e.Verbose {
		log.Printf("%s polling delay: %ds.\n", e.GetName(), e.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", e.GetName(), len(e.EnabledPairs), e.EnabledPairs)
	}

	exchangeProducts, err := e.GetPairSettings()
	if err != nil {
		log.Printf("%s Failed to get available products.\n", e.GetName())
	} else {
		var currencies []string
		for x := range exchangeProducts {
			currencies = append(currencies, x)
		}
		err = e.UpdateAvailableCurrencies(currencies, false)
		if err != nil {
			log.Printf("%s Failed to update available currencies.\n", e.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *EXMO) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	pairsCollated, err := exchange.GetAndFormatExchangeCurrencies(e.Name, e.GetEnabledCurrencies())
	if err != nil {
		return tickerPrice, err
	}

	result, err := e.GetTicker(pairsCollated.String())
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range e.GetEnabledCurrencies() {
		currency := exchange.FormatExchangeCurrency(e.Name, x).String()
		var tickerPrice ticker.Price
		tickerPrice.Pair = x
		tickerPrice.Last = result[currency].Last
		tickerPrice.Ask = result[currency].Sell
		tickerPrice.High = result[currency].High
		tickerPrice.Bid = result[currency].Buy
		tickerPrice.Last = result[currency].Last
		tickerPrice.Low = result[currency].Low
		tickerPrice.Volume = result[currency].Volume
		ticker.ProcessTicker(e.Name, x, tickerPrice, assetType)
	}
	return ticker.GetTicker(e.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (e *EXMO) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tick, err := ticker.GetTicker(e.GetName(), p, assetType)
	if err != nil {
		return e.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// GetOrderbookEx returns the orderbook for a currency pair
func (e *EXMO) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(e.GetName(), p, assetType)
	if err != nil {
		return e.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *EXMO) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	pairsCollated, err := exchange.GetAndFormatExchangeCurrencies(e.Name, e.GetEnabledCurrencies())
	if err != nil {
		return orderBook, err
	}

	orderbookNew, err := e.GetOrderbook(pairsCollated.String())
	if err != nil {
		return orderBook, err
	}

	for _, x := range e.GetEnabledCurrencies() {
		currency := exchange.FormatExchangeCurrency(e.Name, x).String()

		data := orderbookNew[currency]
		for x := range data.Bid {
			obData := data.Bid[x]
			price, _ := strconv.ParseFloat(obData[0], 64)
			amount, _ := strconv.ParseFloat(obData[1], 64)
			orderBook.Bids = append(orderBook.Bids, orderbook.Item{Price: price, Amount: amount})
		}

		for x := range data.Ask {
			obData := data.Ask[x]
			price, _ := strconv.ParseFloat(obData[0], 64)
			amount, _ := strconv.ParseFloat(obData[1], 64)
			orderBook.Asks = append(orderBook.Asks, orderbook.Item{Price: price, Amount: amount})
		}
		orderbook.ProcessOrderbook(e.GetName(), p, orderBook, assetType)
	}

	orderbook.ProcessOrderbook(e.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(e.Name, p, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// Exmo exchange
func (e *EXMO) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = e.GetName()
	result, err := e.GetUserInfo()
	if err != nil {
		return response, err
	}

	for x, y := range result.Balances {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = common.StringToUpper(x)
		for z, w := range result.Reserved {
			if z == x {
				avail, _ := strconv.ParseFloat(y, 64)
				reserved, _ := strconv.ParseFloat(w, 64)
				exchangeCurrency.TotalValue = avail + reserved
				exchangeCurrency.Hold = reserved
			}
		}
		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}
