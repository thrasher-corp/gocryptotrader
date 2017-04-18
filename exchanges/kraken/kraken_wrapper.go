package kraken

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
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
		err = k.UpdateAvailableCurrencies(exchangeProducts)
		if err != nil {
			log.Printf("%s Failed to get config.\n", k.GetName())
		}
	}

	for k.Enabled {
		err := k.GetTicker(common.JoinStrings(k.EnabledPairs, ","))
		if err != nil {
			log.Println(err)
		} else {
			for _, x := range k.EnabledPairs {
				ticker := k.Ticker[x]
				log.Printf("Kraken %s Last %f High %f Low %f Volume %f\n", x, ticker.Last, ticker.High, ticker.Low, ticker.Volume)
				stats.AddExchangeInfo(k.GetName(), x[0:3], x[3:], ticker.Last, ticker.Volume)
			}
		}
		time.Sleep(time.Second * k.RESTPollingDelay)
	}
}

//This will return the TickerPrice struct when tickers are completed here..
func (k *Kraken) GetTickerPrice(p pair.CurrencyPair) (ticker.TickerPrice, error) {
	var tickerPrice ticker.TickerPrice
	/*
		ticker, err := i.GetTicker(currency)
		if err != nil {
			log.Println(err)
			return tickerPrice
		}
		tickerPrice.Ask = ticker.Ask
		tickerPrice.Bid = ticker.Bid
	*/
	return tickerPrice, nil
}

func (k *Kraken) GetOrderbookEx(p pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	return orderbook.OrderbookBase{}, nil
}

//TODO: Retrieve Kraken info
//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the Kraken exchange
func (e *Kraken) GetExchangeAccountInfo() (exchange.ExchangeAccountInfo, error) {
	var response exchange.ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	return response, nil
}
