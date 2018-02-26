package huobi

import (
	"errors"
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the HUOBI go routine
func (h *HUOBI) Start() {
	go h.Run()
}

// Run implements the HUOBI wrapper
func (h *HUOBI) Run() {
	if h.Verbose {
		log.Printf("%s Websocket: %s (url: %s).\n", h.GetName(), common.IsEnabled(h.Websocket), huobiSocketIOAddress)
		log.Printf("%s polling delay: %ds.\n", h.GetName(), h.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", h.GetName(), len(h.EnabledPairs), h.EnabledPairs)
	}

	if h.Websocket {
		go h.WebsocketClient()
	}

	exchangeProducts, err := h.GetSymbols()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", h.GetName())
	} else {
		forceUpgrade := false
		if common.StringDataContains(h.EnabledPairs, "CNY") || common.StringDataContains(h.AvailablePairs, "CNY") {
			forceUpgrade = true
		}

		if common.StringDataContains(h.BaseCurrencies, "CNY") {
			cfg := config.GetConfig()
			exchCfg, errCNY := cfg.GetExchangeConfig(h.Name)
			if err != nil {
				log.Printf("%s failed to get exchange config. %s\n", h.Name, errCNY)
				return
			}
			exchCfg.BaseCurrencies = "USD"
			h.BaseCurrencies = []string{"USD"}

			errCNY = cfg.UpdateExchangeConfig(exchCfg)
			if errCNY != nil {
				log.Printf("%s failed to update config. %s\n", h.Name, errCNY)
				return
			}
		}

		var currencies []string
		for x := range exchangeProducts {
			newCurrency := exchangeProducts[x].BaseCurrency + "-" + exchangeProducts[x].QuoteCurrency
			currencies = append(currencies, newCurrency)
		}

		if forceUpgrade {
			enabledPairs := []string{"btc-usdt"}
			log.Println("WARNING: Available and enabled pairs for Huobi reset due to config upgrade, please enable the ones you would like again")

			err = h.UpdateEnabledCurrencies(enabledPairs, true)
			if err != nil {
				log.Printf("%s Failed to update enabled currencies.\n", h.GetName())
			}
		}
		err = h.UpdateAvailableCurrencies(currencies, forceUpgrade)
		if err != nil {
			log.Printf("%s Failed to update available currencies.\n", h.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (h *HUOBI) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := h.GetMarketDetailMerged(exchange.FormatExchangeCurrency(h.Name, p).String())
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = p
	tickerPrice.Low = tick.Low
	tickerPrice.Last = tick.Close
	tickerPrice.Volume = tick.Volume
	tickerPrice.High = tick.High
	tickerPrice.Ask = tick.Ask[0]
	tickerPrice.Bid = tick.Bid[0]
	ticker.ProcessTicker(h.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(h.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (h *HUOBI) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(h.GetName(), p, assetType)
	if err != nil {
		return h.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (h *HUOBI) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(h.GetName(), p, assetType)
	if err != nil {
		return h.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (h *HUOBI) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := h.GetDepth(exchange.FormatExchangeCurrency(h.Name, p).String(), "step1")
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data[1], Price: data[0]})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data[1], Price: data[0]})
	}

	orderbook.ProcessOrderbook(h.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(h.Name, p, assetType)
}

//GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// HUOBI exchange - to-do
func (h *HUOBI) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = h.GetName()
	return response, nil
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (h *HUOBI) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}
