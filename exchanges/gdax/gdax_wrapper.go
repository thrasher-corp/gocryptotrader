package gdax

import (
	"errors"
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the GDAX go routine
func (g *GDAX) Start() {
	go g.Run()
}

// Run implements the GDAX wrapper
func (g *GDAX) Run() {
	if g.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", g.GetName(), common.IsEnabled(g.Websocket), gdaxWebsocketURL)
		log.Printf("%s polling delay: %ds.\n", g.GetName(), g.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", g.GetName(), len(g.EnabledPairs), g.EnabledPairs)
	}

	if g.Websocket {
		go g.WebsocketClient()
	}

	exchangeProducts, err := g.GetProducts()
	if err != nil {
		log.Printf("%s Failed to get available products.\n", g.GetName())
	} else {
		currencies := []string{}
		for _, x := range exchangeProducts {
			if x.ID != "BTC" && x.ID != "USD" && x.ID != "GBP" {
				currencies = append(currencies, x.ID[0:3]+x.ID[4:])
			}
		}
		err = g.UpdateAvailableCurrencies(currencies, false)
		if err != nil {
			log.Printf("%s Failed to get config.\n", g.GetName())
		}
	}
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// GDAX exchange
func (g *GDAX) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = g.GetName()
	accountBalance, err := g.GetAccounts()
	if err != nil {
		return response, err
	}
	for i := 0; i < len(accountBalance); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = accountBalance[i].Currency
		exchangeCurrency.TotalValue = accountBalance[i].Available
		exchangeCurrency.Hold = accountBalance[i].Hold

		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (g *GDAX) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := g.GetTicker(exchange.FormatExchangeCurrency(g.Name, p).String())
	if err != nil {
		return ticker.Price{}, err
	}

	stats, err := g.GetStats(exchange.FormatExchangeCurrency(g.Name, p).String())

	if err != nil {
		return ticker.Price{}, err
	}

	tickerPrice.Pair = p
	tickerPrice.Volume = stats.Volume
	tickerPrice.Last = tick.Price
	tickerPrice.High = stats.High
	tickerPrice.Low = stats.Low
	ticker.ProcessTicker(g.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(g.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (g *GDAX) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(g.GetName(), p, assetType)
	if err != nil {
		return g.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (g *GDAX) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(g.GetName(), p, assetType)
	if err != nil {
		return g.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (g *GDAX) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := g.GetOrderbook(exchange.FormatExchangeCurrency(g.Name, p).String(), 2)
	if err != nil {
		return orderBook, err
	}

	obNew := orderbookNew.(OrderbookL1L2)

	for x := range obNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: obNew.Bids[x].Amount, Price: obNew.Bids[x].Price})
	}

	for x := range obNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: obNew.Bids[x].Amount, Price: obNew.Bids[x].Price})
	}

	orderbook.ProcessOrderbook(g.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(g.Name, p, assetType)
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (g *GDAX) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}
