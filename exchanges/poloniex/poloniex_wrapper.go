package poloniex

import (
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the Poloniex go routine
func (p *Poloniex) Start() {
	go p.Run()
}

// Run implements the Poloniex wrapper
func (p *Poloniex) Run() {
	if p.Verbose {
		log.Printf("%s Websocket: %s (url: %s).\n", p.GetName(), common.IsEnabled(p.Websocket), POLONIEX_WEBSOCKET_ADDRESS)
		log.Printf("%s polling delay: %ds.\n", p.GetName(), p.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", p.GetName(), len(p.EnabledPairs), p.EnabledPairs)
	}

	if p.Websocket {
		go p.WebsocketClient()
	}

	exchangeCurrencies, err := p.GetExchangeCurrencies()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", p.GetName())
	} else {
		forceUpdate := false
		if common.StringDataCompare(p.AvailablePairs, "BTC_USDT") {
			log.Printf("%s contains invalid pair, forcing upgrade of available currencies.\n",
				p.GetName())
			forceUpdate = true
		}
		err = p.UpdateAvailableCurrencies(exchangeCurrencies, forceUpdate)
		if err != nil {
			log.Printf("%s Failed to update available currencies %s.\n", p.GetName(), err)
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (p *Poloniex) UpdateTicker(currencyPair pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := p.GetTicker()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range p.GetEnabledCurrencies() {
		var tp ticker.Price
		curr := exchange.FormatExchangeCurrency(p.GetName(), x).String()
		tp.Pair = x
		tp.Ask = tick[curr].LowestAsk
		tp.Bid = tick[curr].HighestBid
		tp.High = tick[curr].High24Hr
		tp.Last = tick[curr].Last
		tp.Low = tick[curr].Low24Hr
		tp.Volume = tick[curr].BaseVolume
		ticker.ProcessTicker(p.GetName(), x, tp, assetType)
	}
	return ticker.GetTicker(p.Name, currencyPair, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (p *Poloniex) GetTickerPrice(currencyPair pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(p.GetName(), currencyPair, assetType)
	if err != nil {
		return p.UpdateTicker(currencyPair, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (p *Poloniex) GetOrderbookEx(currencyPair pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(p.GetName(), currencyPair, assetType)
	if err != nil {
		return p.UpdateOrderbook(currencyPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (p *Poloniex) UpdateOrderbook(currencyPair pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := p.GetOrderbook("", 1000)
	if err != nil {
		return orderBook, err
	}

	for _, x := range p.GetEnabledCurrencies() {
		currency := exchange.FormatExchangeCurrency(p.Name, x).String()
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
		orderbook.ProcessOrderbook(p.Name, x, orderBook, assetType)
	}
	return orderbook.GetOrderbook(p.Name, currencyPair, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// Poloniex exchange
func (p *Poloniex) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = p.GetName()
	accountBalance, err := p.GetBalances()
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
