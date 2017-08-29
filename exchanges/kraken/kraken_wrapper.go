package kraken

import (
	"log"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (k *Kraken) Start() {
	go k.Run()
}

func (k *Kraken) Run() {
	if k.Verbose {
		log.Printf("%s polling delay: %ds.\n", k.GetName(), k.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", k.GetName(), len(k.EnabledPairs), k.EnabledPairs)
	}

	assetPairs, err := k.GetAssetPairs()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", k.GetName())
	} else {
		var exchangeProducts []string
		for _, v := range assetPairs {
			exchangeProducts = append(exchangeProducts, v.Altname)
		}
		err = k.UpdateAvailableCurrencies(exchangeProducts, false)
		if err != nil {
			log.Printf("%s Failed to get config.\n", k.GetName())
		}
	}
}

func (k *Kraken) UpdateTicker(p pair.CurrencyPair) (ticker.TickerPrice, error) {
	var tickerPrice ticker.TickerPrice

	pairs := k.GetEnabledCurrencies()
	pairsCollated, err := exchange.GetAndFormatExchangeCurrencies(k.Name, pairs)
	if err != nil {
		return tickerPrice, err
	}
	err = k.GetTicker(pairsCollated.String())
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range pairs {
		var tp ticker.TickerPrice
		tick, ok := k.Ticker[x.Pair().String()]
		if !ok {
			continue
		}

		tp.Pair = x
		tp.Last = tick.Last
		tp.Ask = tick.Ask
		tp.Bid = tick.Bid
		tp.High = tick.High
		tp.Low = tick.Low
		tp.Volume = tick.Volume
		ticker.ProcessTicker(k.GetName(), x, tp)
	}
	return ticker.GetTicker(k.GetName(), p)
}

//This will return the TickerPrice struct when tickers are completed here..
func (k *Kraken) GetTickerPrice(p pair.CurrencyPair) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(k.GetName(), p)
	if err != nil {
		return k.UpdateTicker(p)
	}
	return tickerNew, nil
}

func (k *Kraken) GetOrderbookEx(p pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	return orderbook.OrderbookBase{}, nil
}

//TODO: Retrieve Kraken info
//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the Kraken exchange
func (e *Kraken) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = e.GetName()
	return response, nil
}
