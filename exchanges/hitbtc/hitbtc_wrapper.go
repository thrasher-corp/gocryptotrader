package hitbtc

import (
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the HitBTC go routine
func (p *HitBTC) Start() {
	go p.Run()
}

// Run implements the HitBTC wrapper
func (p *HitBTC) Run() {
	if p.Verbose {
		log.Printf("%s Websocket: %s (url: %s).\n", p.GetName(), common.IsEnabled(p.Websocket), HITBTC_WEBSOCKET_ADDRESS)
		log.Printf("%s polling delay: %ds.\n", p.GetName(), p.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", p.GetName(), len(p.EnabledPairs), p.EnabledPairs)
	}

	if p.Websocket {
		go p.WebsocketClient()
	}

	exchangeProducts, err := p.GetSymbolsDetailed()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", p.GetName())
	} else {
		forceUpgrade := false
		if !common.StringDataContains(p.EnabledPairs, "-") || !common.StringDataContains(p.AvailablePairs, "-") {
			forceUpgrade = true
		}
		var currencies []string
		for x := range exchangeProducts {
			currencies = append(currencies, exchangeProducts[x].BaseCurrency+"-"+exchangeProducts[x].QuoteCurrency)
		}

		if forceUpgrade {
			enabledPairs := []string{"BTC-USD"}
			log.Println("WARNING: Available pairs for HitBTC reset due to config upgrade, please enable the ones you would like again.")

			err = p.UpdateEnabledCurrencies(enabledPairs, true)
			if err != nil {
				log.Printf("%s Failed to update enabled currencies.\n", p.GetName())
			}
		}
		err = p.UpdateAvailableCurrencies(currencies, forceUpgrade)
		if err != nil {
			log.Printf("%s Failed to update available currencies.\n", p.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (p *HitBTC) UpdateTicker(currencyPair pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tick, err := p.GetTicker("")
	if err != nil {
		return ticker.Price{}, err
	}

	for _, x := range p.GetEnabledCurrencies() {
		var tp ticker.Price
		curr := exchange.FormatExchangeCurrency(p.GetName(), x).String()
		tp.Pair = x
		tp.Ask = tick[curr].Ask
		tp.Bid = tick[curr].Bid
		tp.High = tick[curr].High
		tp.Last = tick[curr].Last
		tp.Low = tick[curr].Low
		tp.Volume = tick[curr].Volume
		ticker.ProcessTicker(p.GetName(), x, tp, assetType)
	}
	return ticker.GetTicker(p.Name, currencyPair, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (p *HitBTC) GetTickerPrice(currencyPair pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(p.GetName(), currencyPair, assetType)
	if err != nil {
		return p.UpdateTicker(currencyPair, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (p *HitBTC) GetOrderbookEx(currencyPair pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(p.GetName(), currencyPair, assetType)
	if err != nil {
		return p.UpdateOrderbook(currencyPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (p *HitBTC) UpdateOrderbook(currencyPair pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := p.GetOrderbook(exchange.FormatExchangeCurrency(p.GetName(), currencyPair).String(), 1000)
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

	orderbook.ProcessOrderbook(p.GetName(), currencyPair, orderBook, assetType)
	return orderbook.GetOrderbook(p.Name, currencyPair, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// HitBTC exchange
func (p *HitBTC) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = p.GetName()
	accountBalance, err := p.GetBalances()
	if err != nil {
		return response, err
	}

	for _, item := range accountBalance {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = item.Currency
		exchangeCurrency.TotalValue = item.Available
		exchangeCurrency.Hold = item.Reserved
		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}
