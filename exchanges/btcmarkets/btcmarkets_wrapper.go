package btcmarkets

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (b *BTCMarkets) Start() {
	go b.Run()
}

func (b *BTCMarkets) Run() {
	if b.Verbose {
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	for b.Enabled {
		for _, x := range b.EnabledPairs {
			currency := x
			go func() {
				ticker, err := b.GetTickerPrice(currency)
				if err != nil {
					return
				}
				//BTCMarketsLastUSD, _ := ConvertCurrency(ticker.Last, "AUD", "USD")
				//BTCMarketsBestBidUSD, _ := ConvertCurrency(ticker.Bid, "AUD", "USD")
				//BTCMarketsBestAskUSD, _ := ConvertCurrency(ticker.Ask, "AUD", "USD")
				//log.Printf("BTC Markets %s: Last %f (%f) Bid %f (%f) Ask %f (%f)\n", currency, BTCMarketsLastUSD, ticker.Last, BTCMarketsBestBidUSD, ticker.Bid, BTCMarketsBestAskUSD, ticker.Ask)
				log.Printf("BTC Markets %s: Last %f Bid %f Ask %f \n", currency, ticker.Last, ticker.Bid, ticker.Ask)
				//AddExchangeInfo(b.GetName(), currency[0:3], currency[3:], ticker.Last, 0)
				//AddExchangeInfo(b.GetName(), currency[0:3], "USD", BTCMarketsLastUSD, 0)
			}()
		}
		time.Sleep(time.Second * b.RESTPollingDelay)
	}
}

func (b *BTCMarkets) GetTickerPrice(currency string) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return tickerNew, nil
	}

	var tickerPrice ticker.TickerPrice
	tick, err := b.GetTicker(currency)
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Ask = tick.BestAsk
	tickerPrice.Bid = tick.BestBID
	tickerPrice.FirstCurrency = currency[0:3]
	tickerPrice.SecondCurrency = currency[3:]
	tickerPrice.Last = tick.LastPrice
	ticker.ProcessTicker(b.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the BTCMarkets exchange
func (e *BTCMarkets) GetExchangeAccountInfo() (exchange.ExchangeAccountInfo, error) {
	var response exchange.ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	accountBalance, err := e.GetAccountBalance()
	if err != nil {
		return response, err
	}
	for i := 0; i < len(accountBalance); i++ {
		var exchangeCurrency exchange.ExchangeAccountCurrencyInfo
		exchangeCurrency.CurrencyName = accountBalance[i].Currency
		exchangeCurrency.TotalValue = accountBalance[i].Balance
		exchangeCurrency.Hold = accountBalance[i].PendingFunds

		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}
