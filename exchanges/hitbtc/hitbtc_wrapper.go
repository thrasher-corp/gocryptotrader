package hitbtc

import (
	"errors"
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the HitBTC go routine
func (h *HitBTC) Start() {
	go h.Run()
}

// Run implements the HitBTC wrapper
func (h *HitBTC) Run() {
	if h.Verbose {
		log.Printf("%s Websocket: %s (url: %s).\n", h.GetName(), common.IsEnabled(h.Websocket), hitbtcWebsocketAddress)
		log.Printf("%s polling delay: %ds.\n", h.GetName(), h.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", h.GetName(), len(h.EnabledPairs), h.EnabledPairs)
	}

	if h.Websocket {
		go h.WebsocketClient()
	}

	exchangeProducts, err := h.GetSymbolsDetailed()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", h.GetName())
	} else {
		forceUpgrade := false
		if !common.StringDataContains(h.EnabledPairs, "-") || !common.StringDataContains(h.AvailablePairs, "-") {
			forceUpgrade = true
		}
		var currencies []string
		for x := range exchangeProducts {
			currencies = append(currencies, exchangeProducts[x].BaseCurrency+"-"+exchangeProducts[x].QuoteCurrency)
		}

		if forceUpgrade {
			enabledPairs := []string{"BTC-USD"}
			log.Println("WARNING: Available pairs for HitBTC reset due to config upgrade, please enable the ones you would like again.")

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
func (h *HitBTC) UpdateTicker(currencyPair pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tick, err := h.GetTicker("")
	if err != nil {
		return ticker.Price{}, err
	}

	for _, x := range h.GetEnabledCurrencies() {
		var tp ticker.Price
		curr := exchange.FormatExchangeCurrency(h.GetName(), x).String()
		tp.Pair = x
		tp.Ask = tick[curr].Ask
		tp.Bid = tick[curr].Bid
		tp.High = tick[curr].High
		tp.Last = tick[curr].Last
		tp.Low = tick[curr].Low
		tp.Volume = tick[curr].Volume
		ticker.ProcessTicker(h.GetName(), x, tp, assetType)
	}
	return ticker.GetTicker(h.Name, currencyPair, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (h *HitBTC) GetTickerPrice(currencyPair pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(h.GetName(), currencyPair, assetType)
	if err != nil {
		return h.UpdateTicker(currencyPair, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (h *HitBTC) GetOrderbookEx(currencyPair pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(h.GetName(), currencyPair, assetType)
	if err != nil {
		return h.UpdateOrderbook(currencyPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (h *HitBTC) UpdateOrderbook(currencyPair pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := h.GetOrderbook(exchange.FormatExchangeCurrency(h.GetName(), currencyPair).String(), 1000)
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

	orderbook.ProcessOrderbook(h.GetName(), currencyPair, orderBook, assetType)
	return orderbook.GetOrderbook(h.Name, currencyPair, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// HitBTC exchange
func (h *HitBTC) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = h.GetName()
	accountBalance, err := h.GetBalances()
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

// GetExchangeHistory returns historic trade data since exchange opening.
func (h *HitBTC) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}
