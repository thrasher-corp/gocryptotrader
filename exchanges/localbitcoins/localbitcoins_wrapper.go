package localbitcoins

import (
	"log"

	"github.com/thrasher-/gocryptotrader/common"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (l *LocalBitcoins) Start() {
	go l.Run()
}

func (l *LocalBitcoins) Run() {
	if l.Verbose {
		log.Printf("%s polling delay: %ds.\n", l.GetName(), l.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", l.GetName(), len(l.EnabledPairs), l.EnabledPairs)
	}
}

func (l *LocalBitcoins) UpdateTicker(p pair.CurrencyPair) (ticker.TickerPrice, error) {
	var tickerPrice ticker.TickerPrice
	tick, err := l.GetTicker()
	if err != nil {
		return tickerPrice, err
	}

	for key, value := range tick {
		currency := pair.NewCurrencyPair("BTC", common.StringToUpper(key))
		var tp ticker.TickerPrice
		tp.Pair = currency
		tp.Last = value.Rates.Last
		tp.Volume = value.VolumeBTC
		ticker.ProcessTicker(l.GetName(), currency, tp)
	}

	return ticker.GetTicker(l.GetName(), p)
}

func (l *LocalBitcoins) GetTickerPrice(p pair.CurrencyPair) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(l.GetName(), p)
	if err == nil {
		return l.UpdateTicker(p)
	}
	return tickerNew, nil
}

func (l *LocalBitcoins) GetOrderbookEx(p pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	return orderbook.OrderbookBase{}, nil
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the LocalBitcoins exchange
func (e *LocalBitcoins) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = e.GetName()
	accountBalance, err := e.GetWalletBalance()
	if err != nil {
		return response, err
	}
	var exchangeCurrency exchange.AccountCurrencyInfo
	exchangeCurrency.CurrencyName = "BTC"
	exchangeCurrency.TotalValue = accountBalance.Total.Balance

	response.Currencies = append(response.Currencies, exchangeCurrency)
	return response, nil
}
